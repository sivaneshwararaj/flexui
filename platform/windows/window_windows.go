//go:build windows

package windows

import (
	"fmt"
	"image"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/sivaneshwararaj/flexui/platform"
)

// Window is one native window showing an AppHandler's frames.
type Window struct {
	app       platform.AppHandler
	hwnd      uintptr
	buf       *image.RGBA // the app's frame (RGBA, premultiplied)
	bgra      []byte      // the same pixels in GDI's BGRA order
	w, h      int         // logical size
	scale     float64     // DPI / 96
	frames    int
	tracking  bool   // TrackMouseEvent armed for WM_MOUSELEAVE
	surr      uint16 // pending high surrogate from WM_CHAR
	redrawing bool
	// OnFrame, if set, is called after each rendered frame.
	OnFrame func(img *image.RGBA, w, h int, scale float64)
}

// the current window: the window procedure finds it here (one window per
// process).
var current *Window

var (
	classReady bool
	wndProcCB  uintptr
	className  = windows.StringToUTF16Ptr("FlexUIWindow")
)

func registerClass() error {
	if classReady {
		return nil
	}
	wndProcCB = windows.NewCallback(wndProc)
	hinst, _, _ := pGetModuleHandleW.Call(0)
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	wc := wndClassExW{
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   wndProcCB,
		hInstance:     hinst,
		hCursor:       cursor,
		lpszClassName: className,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if atom, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return fmt.Errorf("windows: RegisterClassExW: %v", err)
	}
	classReady = true
	debugf("window class registered")
	return nil
}

// Open creates a window for app, shows it and renders the first frame,
// without running the message loop.
func Open(cfg platform.Config, app platform.AppHandler) (*Window, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	if r, _, err := pSetProcessDpiAwarenessContext.Call(dpiAwarenessPerMonitorV2); r == 0 {
		debugf("SetProcessDpiAwarenessContext: %v (already set, or older Windows)", err)
	}
	if err := registerClass(); err != nil {
		return nil, err
	}
	w := &Window{app: app}
	current = w
	title := windows.StringToUTF16Ptr(cfg.Title)
	hwnd, _, err := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWin, cwUseDefault, cwUseDefault, cwUseDefault, cwUseDefault, 0, 0, 0, 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("windows: CreateWindowExW: %v", err)
	}
	w.hwnd = hwnd
	// Size the client area to cfg in logical pixels at this monitor's DPI.
	dpi, _, _ := pGetDpiForWindow.Call(hwnd)
	if dpi == 0 {
		dpi = 96
	}
	r := rect{Right: int32(float64(cfg.Width)*float64(dpi)/96 + 0.5), Bottom: int32(float64(cfg.Height)*float64(dpi)/96 + 0.5)}
	pAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&r)), wsOverlappedWin, 0, 0, dpi)
	pSetWindowPos.Call(hwnd, 0, 0, 0, uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpNoZOrder|swpNoActivate|swpNoMove)
	debugf("window %#x, dpi %d, outer %dx%d", hwnd, dpi, r.Right-r.Left, r.Bottom-r.Top)
	pShowWindow.Call(hwnd, swShow)
	pUpdateWindow.Call(hwnd)
	w.render()
	return w, nil
}

// Loop runs the message loop until the window is closed.
func (w *Window) Loop() {
	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 { // 0: WM_QUIT, -1: error
			debugf("message loop ends (%d)", int32(r))
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// Size reports the client area in logical pixels and the DPI scale.
func (w *Window) Size() (width, height int, scale float64) {
	var r rect
	pGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	dpi, _, _ := pGetDpiForWindow.Call(w.hwnd)
	if dpi == 0 {
		dpi = 96
	}
	scale = float64(dpi) / 96
	return int(float64(r.Right-r.Left)/scale + 0.5), int(float64(r.Bottom-r.Top)/scale + 0.5), scale
}

// LastFrame returns the last rendered frame and its logical size and
// scale.
func (w *Window) LastFrame() (*image.RGBA, int, int, float64) { return w.buf, w.w, w.h, w.scale }

// Redraw renders a new frame now. It does nothing when called from
// within OnFrame.
func (w *Window) Redraw() {
	if w.redrawing {
		return
	}
	w.redrawing = true
	defer func() { w.redrawing = false }()
	w.render()
}

// render asks the app for a frame, converts it to BGRA and invalidates
// the window so that WM_PAINT draws it.
func (w *Window) render() {
	t0 := time.Now()
	lw, lh, scale := w.Size()
	if lw <= 0 || lh <= 0 {
		return // minimized
	}
	if lw != w.w || lh != w.h {
		if w.w != 0 {
			w.app.Event(platform.Event{Kind: platform.EventResize, Width: lw, Height: lh})
		}
		w.w, w.h = lw, lh
	}
	w.scale = scale
	pw, ph := int(float64(lw)*scale+0.5), int(float64(lh)*scale+0.5)
	if w.buf == nil || w.buf.Rect.Dx() != pw || w.buf.Rect.Dy() != ph {
		w.buf = image.NewRGBA(image.Rect(0, 0, pw, ph))
		w.bgra = make([]byte, len(w.buf.Pix))
	}
	w.app.Frame(w.buf, lw, lh, scale)
	src, dst := w.buf.Pix, w.bgra
	for i := 0; i < len(src); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
	pInvalidateRect.Call(w.hwnd, 0, 0)
	w.frames++
	debugf("frame %d: %dx%d @%gx (%dx%d px) in %v", w.frames, lw, lh, scale, pw, ph, time.Since(t0))
	if w.OnFrame != nil {
		w.OnFrame(w.buf, lw, lh, scale)
	}
}

// paint draws the BGRA frame at the top-left of the client area.
func (w *Window) paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
	if w.buf != nil && len(w.bgra) > 0 {
		pw, ph := w.buf.Rect.Dx(), w.buf.Rect.Dy()
		bi := bitmapInfo{header: bitmapInfoHeader{
			biWidth: int32(pw), biHeight: -int32(ph), // negative: top-down rows
			biPlanes: 1, biBitCount: 32, biCompression: biRGB,
		}}
		bi.header.biSize = uint32(unsafe.Sizeof(bi.header))
		pSetDIBitsToDevice.Call(hdc, 0, 0, uintptr(pw), uintptr(ph), 0, 0, 0, uintptr(ph),
			uintptr(unsafe.Pointer(&w.bgra[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors)
	}
	pEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
}

func (w *Window) send(e platform.Event) {
	debugf("event %+v", e)
	w.app.Event(e)
	w.render()
}

// Coordinates in lParam are signed 16-bit client pixels.
func xy(lParam uintptr) (int32, int32) {
	return int32(int16(lParam & 0xFFFF)), int32(int16((lParam >> 16) & 0xFFFF))
}

func (w *Window) pointer(kind platform.EventKind, lParam uintptr) {
	x, y := xy(lParam)
	if kind == platform.EventPointerMove && !w.tracking {
		tme := trackMouseEvent{dwFlags: tmeLeave, hwndTrack: w.hwnd}
		tme.cbSize = uint32(unsafe.Sizeof(tme))
		pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		w.tracking = true
	}
	w.send(platform.Event{Kind: kind, X: float64(x) / w.scale, Y: float64(y) / w.scale})
}

// wheelStep is the scroll distance of one wheel notch (120 units): three
// lines of 16 logical pixels, the Windows default of 3 lines.
const wheelStep = 48.0

func (w *Window) wheel(wParam, lParam uintptr, horizontal bool) {
	delta := float64(int16((wParam >> 16) & 0xFFFF))
	x, y := xy(lParam)
	p := point{x, y} // screen coordinates for wheel messages
	pScreenToClient.Call(w.hwnd, uintptr(unsafe.Pointer(&p)))
	e := platform.Event{Kind: platform.EventWheel, X: float64(p.X) / w.scale, Y: float64(p.Y) / w.scale}
	if horizontal {
		e.DX = delta / 120 * wheelStep // positive: tilt right
	} else {
		e.DY = -delta / 120 * wheelStep // positive delta: wheel away from the user, scroll up
	}
	w.send(e)
}

func keyDown(vk uintptr) bool {
	r, _, _ := pGetKeyState.Call(vk)
	return r&0x8000 != 0
}

func (w *Window) mods() uint8 {
	var m uint8
	if keyDown(vkShift) {
		m |= platform.ModShift
	}
	if keyDown(vkControl) {
		m |= platform.ModCtrl
	}
	if keyDown(vkMenu) {
		m |= platform.ModAlt
	}
	return m
}

func (w *Window) key(vk uintptr, down bool) {
	name := keyName(vk)
	if name == "" {
		return
	}
	kind := platform.EventKeyUp
	if down {
		kind = platform.EventKeyDown
	}
	w.send(platform.Event{Kind: kind, Key: name, Mods: w.mods()})
}

// char handles WM_CHAR: UTF-16 units (surrogate pairs arrive in two
// messages); control characters (Ctrl+letter, Backspace, Tab, Enter) are
// not text.
func (w *Window) char(u uint16) {
	switch {
	case utf16.IsSurrogate(rune(u)) && u < 0xDC00:
		w.surr = u
		return
	case utf16.IsSurrogate(rune(u)):
		r := utf16.DecodeRune(rune(w.surr), rune(u))
		w.surr = 0
		w.send(platform.Event{Kind: platform.EventText, Text: string(r)})
	case u < 0x20 || u == 0x7F:
		return
	default:
		w.send(platform.Event{Kind: platform.EventText, Text: string(rune(u))})
	}
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := current
	if w == nil || hwnd != w.hwnd {
		r, _, _ := pDefWindowProcW.Call(hwnd, message, wParam, lParam)
		return r
	}
	switch message {
	case wmPaint:
		w.paint()
		return 0
	case wmEraseBkgnd:
		return 1 // the frame covers everything: no flicker
	case wmSize:
		debugf("WM_SIZE %dx%d", lParam&0xFFFF, (lParam>>16)&0xFFFF)
		w.render()
		return 0
	case wmDpiChanged:
		// Move/resize to the suggested rectangle; WM_SIZE re-renders.
		r := (*rect)(ptr(lParam))
		debugf("WM_DPICHANGED dpi %d", wParam&0xFFFF)
		pSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpNoZOrder|swpNoActivate)
		w.render()
		return 0
	case wmMouseMove:
		w.pointer(platform.EventPointerMove, lParam)
		return 0
	case wmMouseLeave:
		w.tracking = false
		w.send(platform.Event{Kind: platform.EventPointerLeave})
		return 0
	case wmLButtonDown:
		pSetCapture.Call(hwnd)
		w.pointer(platform.EventPointerDown, lParam)
		return 0
	case wmLButtonUp:
		pReleaseCapture.Call()
		w.pointer(platform.EventPointerUp, lParam)
		return 0
	case wmMouseWheel:
		w.wheel(wParam, lParam, false)
		return 0
	case wmMouseHWheel:
		w.wheel(wParam, lParam, true)
		return 0
	case wmKeyDown, wmSysKeyDown:
		w.key(wParam, true)
		if message == wmSysKeyDown {
			break // let Alt+F4 and the system menu work
		}
		return 0
	case wmKeyUp, wmSysKeyUp:
		w.key(wParam, false)
		if message == wmSysKeyUp {
			break
		}
		return 0
	case wmChar:
		w.char(uint16(wParam))
		return 0
	case wmClose:
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		debugf("WM_DESTROY")
		w.app.Event(platform.Event{Kind: platform.EventClose})
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

// Clipboard returns the clipboard owned by this window.
func (w *Window) Clipboard() Clipboard { return Clipboard{hwnd: w.hwnd} }

// App is what Run needs (*flexui.App satisfies it).
type App interface {
	platform.AppHandler
	Config() platform.Config
}

type clipboardUser interface{ SetClipboard(platform.Clipboard) }

// Run opens a window for app (its Config gives the title and size), gives
// it the Windows clipboard, and runs the message loop until the window is
// closed. Call it from the main goroutine.
func Run(app App) error { return Platform{}.Run(app.Config(), app) }

// Platform implements platform.Platform.
type Platform struct{}

// Run implements platform.Platform.
func (Platform) Run(cfg platform.Config, app platform.AppHandler) error {
	w, err := Open(cfg, app)
	if err != nil {
		return err
	}
	if cu, ok := app.(clipboardUser); ok {
		cu.SetClipboard(Clipboard{hwnd: w.hwnd})
	}
	w.Loop()
	return nil
}
