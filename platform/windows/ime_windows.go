//go:build windows

package windows

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sivaneshwararaj/flexui/platform"
)

// IME composition through IMM32 (spec/SUBSET.md §8.7). The marked text
// is drawn by flexui in the input (the IME's own composition window is
// hidden); the candidate window is placed below the caret.

var (
	imm32                    = windows.NewLazySystemDLL("imm32.dll")
	pImmGetContext           = imm32.NewProc("ImmGetContext")
	pImmReleaseContext       = imm32.NewProc("ImmReleaseContext")
	pImmGetCompositionString = imm32.NewProc("ImmGetCompositionStringW")
	pImmSetCandidateWindow   = imm32.NewProc("ImmSetCandidateWindow")
	pImmSetCompositionWindow = imm32.NewProc("ImmSetCompositionWindow")
)

const (
	wmImeSetContext            = 0x0281
	wmImeStartComposition      = 0x010D
	wmImeEndComposition        = 0x010E
	wmImeComposition           = 0x010F
	gcsCompStr                 = 0x0008
	gcsCursorPos               = 0x0080
	gcsResultStr               = 0x0800
	iscShowUICompositionWindow = 0x80000000
	cfsPoint                   = 0x0002
	cfsCandidatePos            = 0x0040
	cfsExclude                 = 0x0080
)

type compositionForm struct {
	dwStyle      uint32
	ptCurrentPos point
	rcArea       rect
}

type candidateForm struct {
	dwIndex      uint32
	dwStyle      uint32
	ptCurrentPos point
	rcArea       rect
}

// imeProcs are resolved by stage 1 with the others.
var imeProcs = []*windows.LazyProc{
	pImmGetContext, pImmReleaseContext, pImmGetCompositionString,
	pImmSetCandidateWindow, pImmSetCompositionWindow,
}

// compString reads one of the composition strings (GCS_COMPSTR,
// GCS_RESULTSTR) as UTF-16 units.
func compString(himc, which uintptr) []uint16 {
	n, _, _ := pImmGetCompositionString.Call(himc, which, 0, 0)
	if int32(n) <= 0 {
		return nil
	}
	buf := make([]uint16, n/2)
	pImmGetCompositionString.Call(himc, which, uintptr(unsafe.Pointer(&buf[0])), n)
	return buf
}

// ime handles the IME messages. It reports whether the message was
// handled (and DefWindowProc must not see it).
func (w *Window) ime(message, wParam, lParam uintptr) (uintptr, bool) {
	switch message {
	case wmImeSetContext:
		// Hide the IME's own composition window: flexui draws the marked
		// text in the input.
		r, _, _ := pDefWindowProcW.Call(w.hwnd, message, wParam, lParam&^iscShowUICompositionWindow)
		return r, true
	case wmImeStartComposition:
		debugf("WM_IME_STARTCOMPOSITION")
		w.composing = true
		w.placeCandidates()
		return 0, true
	case wmImeComposition:
		himc, _, _ := pImmGetContext.Call(w.hwnd)
		if himc == 0 {
			return 0, false
		}
		defer pImmReleaseContext.Call(w.hwnd, himc)
		if lParam&gcsResultStr != 0 {
			s, _ := imeText(compString(himc, gcsResultStr), 0)
			debugf("WM_IME_COMPOSITION result %q", s)
			w.send(platform.Event{Kind: platform.EventComposeCommit, Text: s})
			w.composing = false
		}
		if lParam&gcsCompStr != 0 {
			u := compString(himc, gcsCompStr)
			cur := len(u)
			if lParam&gcsCursorPos != 0 {
				c, _, _ := pImmGetCompositionString.Call(himc, gcsCursorPos, 0, 0)
				cur = int(int32(c))
			}
			s, at := imeText(u, cur)
			debugf("WM_IME_COMPOSITION marked %q cursor %d", s, at)
			if s == "" && w.composing {
				w.send(platform.Event{Kind: platform.EventComposeCancel})
			} else if s != "" {
				w.composing = true
				w.send(platform.Event{Kind: platform.EventCompose, Text: s, SelStart: at, SelEnd: at})
			}
			w.placeCandidates()
		}
		return 0, true
	case wmImeEndComposition:
		debugf("WM_IME_ENDCOMPOSITION (composing %v)", w.composing)
		if w.composing { // ended without a result: cancelled (Escape)
			w.send(platform.Event{Kind: platform.EventComposeCancel})
			w.composing = false
		}
		return 0, true
	}
	return 0, false
}

// placeCandidates puts the IME candidate window below the focused input's
// caret (physical client pixels), excluding the caret's line.
func (w *Window) placeCandidates() {
	cr, ok := w.app.(platform.CaretRecter)
	if !ok {
		return
	}
	x, y, cw, h, ok := cr.CaretRect()
	if !ok {
		return
	}
	himc, _, _ := pImmGetContext.Call(w.hwnd)
	if himc == 0 {
		return
	}
	defer pImmReleaseContext.Call(w.hwnd, himc)
	px := func(v float64) int32 { return int32(v*w.scale + 0.5) }
	caret := rect{px(x), px(y), px(x + cw), px(y + h)}
	cf := compositionForm{dwStyle: cfsPoint, ptCurrentPos: point{caret.Left, caret.Top}}
	pImmSetCompositionWindow.Call(himc, uintptr(unsafe.Pointer(&cf)))
	cand := candidateForm{dwIndex: 0, dwStyle: cfsExclude, ptCurrentPos: point{caret.Left, caret.Bottom}, rcArea: caret}
	pImmSetCandidateWindow.Call(himc, uintptr(unsafe.Pointer(&cand)))
	debugf("candidate window at %d,%d (caret %v)", caret.Left, caret.Bottom, caret)
}
