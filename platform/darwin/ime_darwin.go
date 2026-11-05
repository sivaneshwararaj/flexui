//go:build darwin

package darwin

import (
	"runtime"

	"github.com/ebitengine/purego/objc"

	"github.com/sivaneshwararaj/flexui/platform"
)

// IME composition through NSTextInputClient (spec/SUBSET.md §8.7).
//
// purego callbacks cannot return structs, but three of the protocol's
// methods return NSRange or NSRect. Those are small assembly functions
// (ime_darwin_arm64.s, ime_darwin_amd64.s) that return values kept in the
// globals below, which the Go side updates whenever the composition or
// the caret changes. NSRange arguments arrive as two integer registers on
// both architectures, so the Go callbacks take them as two uint64.

var (
	imeMarked   [2]uint64  // markedRange: location, length (NSNotFound when none)
	imeSelected [2]uint64  // selectedRange
	imeRect     [4]float64 // firstRectForCharacterRange: screen x, y, w, h
)

const nsNotFound = 1<<63 - 1

// Assembly method implementations and their addresses (ABI0).
func markedRangeIMP()
func selectedRangeIMP()
func firstRectIMP()
func imeIMPs() (marked, selected, rect uintptr)

const (
	encRange = "{_NSRange=QQ}16@0:8"
	encRect  = "{CGRect={CGPoint=dd}{CGSize=dd}}40@0:8{_NSRange=QQ}16^{_NSRange=QQ}32"
)

// addTextInputClient adds the NSTextInputClient methods to the view class.
func addTextInputClient(c objc.Class) error {
	if p := objc.GetProtocol("NSTextInputClient"); p != nil {
		if !c.AddProtocol(p) {
			debugf("NSTextInputClient: AddProtocol failed")
		}
	} else {
		debugf("NSTextInputClient protocol not found")
	}
	m, s, r := imeIMPs()
	ok := c.AddMethod(sel("markedRange"), objc.IMP(m), encRange) &&
		c.AddMethod(sel("selectedRange"), objc.IMP(s), encRange) &&
		c.AddMethod(sel("firstRectForCharacterRange:actualRange:"), objc.IMP(r), encRect)
	add := func(name string, fn any, enc string) {
		if !c.AddMethod(sel(name), objc.NewIMP(fn), enc) {
			ok = false
		}
	}
	add("insertText:replacementRange:", func(self objc.ID, _ objc.SEL, text objc.ID, loc, n uint64) {
		if current != nil {
			current.imeInsert(nsText(text))
		}
	}, "v@:@{_NSRange=QQ}")
	add("setMarkedText:selectedRange:replacementRange:", func(self objc.ID, _ objc.SEL, text objc.ID, sl, sn, rl, rn uint64) {
		if current != nil {
			current.imeSetMarked(nsText(text), int(sl))
		}
	}, "v@:@{_NSRange=QQ}{_NSRange=QQ}")
	add("unmarkText", func(self objc.ID, _ objc.SEL) {
		if current != nil {
			current.imeUnmark()
		}
	}, "v@:")
	boolEnc := "B@:" // BOOL is bool on arm64, signed char on x86_64
	if runtime.GOARCH == "amd64" {
		boolEnc = "c@:"
	}
	add("hasMarkedText", func(self objc.ID, _ objc.SEL) bool {
		return current != nil && current.marked != ""
	}, boolEnc)
	add("validAttributesForMarkedText", func(self objc.ID, _ objc.SEL) objc.ID {
		return class("NSArray").Send(sel("array"))
	}, "@@:")
	add("attributedSubstringForProposedRange:actualRange:", func(self objc.ID, _ objc.SEL, loc, n uint64, actual uintptr) objc.ID {
		return 0 // nil: no access to the document's text
	}, "@@:{_NSRange=QQ}^{_NSRange=QQ}")
	add("characterIndexForPoint:", func(self objc.ID, _ objc.SEL, x, y float64) uint64 {
		return nsNotFound
	}, "Q@:{CGPoint=dd}")
	add("doCommandBySelector:", func(self objc.ID, _ objc.SEL, cmd objc.SEL) {
		// Keys are delivered as EventKeyDown before interpretKeyEvents.
	}, "v@::")
	if !ok {
		debugf("NSTextInputClient: some methods could not be added")
	}
	imeMarked = [2]uint64{nsNotFound, 0}
	return nil
}

// nsText returns the string of an NSString or NSAttributedString.
func nsText(t objc.ID) string {
	if t == 0 {
		return ""
	}
	if objc.Send[bool](t, sel("isKindOfClass:"), class("NSAttributedString")) {
		t = t.Send(sel("string"))
	}
	return goString(t)
}

func (w *Window) imeInsert(s string) {
	debugf("insertText %q (marked %q)", s, w.marked)
	if w.marked != "" {
		w.marked = ""
		w.send(platform.Event{Kind: platform.EventComposeCommit, Text: s})
	} else if s != "" {
		w.send(platform.Event{Kind: platform.EventText, Text: s})
	}
	w.imeState()
}

func (w *Window) imeSetMarked(s string, selLoc int) {
	debugf("setMarkedText %q sel %d", s, selLoc)
	switch {
	case s == "" && w.marked != "":
		w.marked = ""
		w.send(platform.Event{Kind: platform.EventComposeCancel})
	case s != "":
		w.marked = s
		at := utf16ToRune(s, selLoc)
		w.send(platform.Event{Kind: platform.EventCompose, Text: s, SelStart: at, SelEnd: at})
	}
	w.imeState()
}

func (w *Window) imeUnmark() {
	debugf("unmarkText (marked %q)", w.marked)
	if w.marked != "" {
		s := w.marked
		w.marked = ""
		w.send(platform.Event{Kind: platform.EventComposeCommit, Text: s})
	}
	w.imeState()
}

// imeState updates the values the assembly methods return: the marked
// range and the caret rectangle in screen coordinates.
func (w *Window) imeState() {
	if w.marked != "" {
		imeMarked = [2]uint64{0, uint64(utf16Len(w.marked))}
		imeSelected = [2]uint64{uint64(utf16Len(w.marked)), 0}
	} else {
		imeMarked, imeSelected = [2]uint64{nsNotFound, 0}, [2]uint64{0, 0}
	}
	cr, ok := w.app.(platform.CaretRecter)
	if !ok {
		return
	}
	x, y, cw, h, ok := cr.CaretRect()
	if !ok {
		return
	}
	// The view is flipped (y down): convert to window, then to screen.
	r := objc.Send[NSRect](w.view, sel("convertRect:toView:"), NSRect{Origin: NSPoint{x, y}, Size: NSSize{cw, h}}, objc.ID(0))
	r = objc.Send[NSRect](w.win, sel("convertRectToScreen:"), r)
	imeRect = [4]float64{r.Origin.X, r.Origin.Y, r.Size.Width, r.Size.Height}
	debugf("caret rect view %.1f,%.1f %.1fx%.1f screen %v", x, y, cw, h, imeRect)
}
