package view

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/edit"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/render"
)

// Clipboard is the text clipboard used for copy, cut and paste.
type Clipboard interface {
	ReadText() (string, error)
	WriteText(s string) error
}

type memClipboard struct{ s string }

func (m *memClipboard) ReadText() (string, error) { return m.s, nil }
func (m *memClipboard) WriteText(s string) error  { m.s = s; return nil }

// Modifier bits (the same values as platform.Mod*).
const (
	ModShift uint8 = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// inputState is the editing state of one <input>, kept across rebuilds
// by element key (spec/SUBSET.md §8.6).
type inputState struct {
	buf     *edit.Buffer
	attr    string    // the value attribute at the last render
	scrollX geom.Unit // horizontal scroll of the text
	atFocus string    // value when focused (for change events)
	// comp is the IME composition in progress: the marked text is in the
	// buffer at [start, start+n) (as in browsers, it is part of the value
	// while composing).
	comp struct {
		active   bool
		start, n int
	}
}

// syncInputs creates, updates and drops input states for a new document:
// a value attribute that changed since the last render (and differs from
// the text) replaces the text; otherwise the user's edits stay.
func (v *View) syncInputs() {
	seen := map[string]bool{}
	var walk func(n *dom.Node)
	walk = func(n *dom.Node) {
		if n.Kind != dom.ElementNode {
			return
		}
		if n.Tag() == "input" {
			k := key(n)
			seen[k] = true
			attr := n.Attr("value")
			st := v.inputs[k]
			switch {
			case st == nil:
				v.inputs[k] = &inputState{buf: edit.New(attr), attr: attr, atFocus: attr}
			case attr != st.attr:
				st.attr = attr
				if attr != st.buf.String() {
					st.buf.Set(attr)
					st.scrollX = 0
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(v.parsed.Dom.Root)
	for k := range v.inputs {
		if !seen[k] {
			delete(v.inputs, k)
		}
	}
}

func (v *View) input(n *dom.Node) *inputState {
	if n == nil || n.Tag() != "input" {
		return nil
	}
	return v.inputs[key(n)]
}

// inputView is the render hook: what to draw in an input.
func (v *View) inputView(n *dom.Node) render.InputState {
	st := v.input(n)
	if st == nil {
		return render.InputState{}
	}
	rs := render.InputState{
		Text: st.buf.String(), Caret: st.buf.Caret, Anchor: st.buf.Anchor,
		Focused: n == v.focus, ScrollX: st.scrollX,
	}
	if st.comp.active {
		rs.MarkStart, rs.MarkEnd = st.comp.start, st.comp.start+st.comp.n
	}
	return rs
}

// InputValue returns the current text of the input with the given id.
func (v *View) InputValue(id string) (string, bool) {
	if v.parsed == nil {
		return "", false
	}
	n := findKey(v.parsed.Dom.Root, "#"+id)
	if st := v.input(n); st != nil {
		return st.buf.String(), true
	}
	return "", false
}

// SetClipboard sets the clipboard used by inputs (default: in memory).
func (v *View) SetClipboard(c Clipboard) { v.clip = c }

// reveal scrolls the input's text so that the caret is visible, and
// repaints.
func (v *View) reveal(n *dom.Node, st *inputState) {
	v.needPaint = true
	if v.doc == nil {
		return
	}
	s := st.buf.String()
	x, w := v.doc.InputCaretX(n, s, st.buf.Caret)
	room := w - geom.Px(1) // the caret is 1px wide
	switch {
	case x-st.scrollX > room:
		st.scrollX = x - room
	case x < st.scrollX:
		st.scrollX = x
	}
	lo, hi := v.doc.InputScrollRange(n, s, room)
	st.scrollX = min(max(st.scrollX, lo), hi)
}

// edited sends an input event after a change of the text.
func (v *View) edited(n *dom.Node, st *inputState) {
	v.reveal(n, st)
	v.dispatch(&Event{Type: "input", Target: n, Value: st.buf.String()})
}

// changed sends a change event if the value differs from when it was
// focused (or last changed), as on blur and Enter in browsers.
func (v *View) changed(n *dom.Node, st *inputState) {
	if s := st.buf.String(); s != st.atFocus {
		st.atFocus = s
		v.dispatch(&Event{Type: "change", Target: n, Value: s})
	}
}

// editKey is the default action of a key in a focused input. It reports
// whether the key was handled.
func (v *View) editKey(n *dom.Node, st *inputState, k string, mods uint8) bool {
	b := st.buf
	shift := mods&ModShift != 0
	cmd := mods&(ModCtrl|ModMeta) != 0
	unit := edit.Char
	switch {
	case mods&ModMeta != 0:
		unit = edit.Line
	case mods&(ModAlt|ModCtrl) != 0:
		unit = edit.Word
	}
	switch k {
	case "ArrowLeft", "ArrowRight":
		// Logical movement, as Chromium: Right is forward in a
		// left-to-right input, backward in a right-to-left one.
		b.Move((k == "ArrowRight") != v.doc.InputRTL(n), unit, shift)
	case "Home", "End", "ArrowUp", "ArrowDown":
		b.Move(k == "End" || k == "ArrowDown", edit.Line, shift)
	case "Backspace", "Delete":
		if b.Delete(k == "Delete", unit) {
			v.edited(n, st)
			return true
		}
	case "Enter":
		v.changed(n, st)
	case "a", "A":
		if !cmd {
			return false
		}
		b.SelectAll()
	case "c", "C", "x", "X":
		if !cmd {
			return false
		}
		if sel := b.Selected(); sel != "" {
			v.clip.WriteText(sel)
			if k == "x" || k == "X" {
				b.Delete(false, edit.Char)
				v.edited(n, st)
				return true
			}
		}
	case "v", "V":
		if !cmd {
			return false
		}
		if s, err := v.clip.ReadText(); err == nil && b.Insert(s) {
			v.edited(n, st)
			return true
		}
	default:
		return false
	}
	v.reveal(n, st)
	return true
}

// Text inserts committed text (typing, IME commit) into the focused
// input. It reports whether the text changed.
func (v *View) Text(s string) bool {
	if v.Document() == nil {
		return false
	}
	n := v.focus
	st := v.input(n)
	if st == nil || n.Disabled() {
		return false
	}
	if !st.buf.Insert(s) {
		return false
	}
	v.edited(n, st)
	return true
}

// indexAt returns the grapheme boundary of input n nearest to document
// x coordinate x.
func (v *View) indexAt(n *dom.Node, st *inputState, x geom.Unit) int {
	cb := v.doc.InputContent(n)
	rx := x - cb.X + st.scrollX
	s := st.buf.String()
	best, bestD := 0, geom.Unit(-1)
	for _, i := range st.buf.Boundaries() {
		cx, _ := v.doc.InputCaretX(n, s, i)
		d := abs(cx - rx)
		if bestD < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// InputSelection returns the caret and anchor of the input with the
// given id (for tests).
func (v *View) InputSelection(id string) (caret, anchor int) {
	if v.parsed == nil {
		return 0, 0
	}
	if st := v.input(findKey(v.parsed.Dom.Root, "#"+id)); st != nil {
		return st.buf.Caret, st.buf.Anchor
	}
	return 0, 0
}

// Compose starts or updates an IME composition in the focused input: the
// marked text replaces the previous marked text (or, when a composition
// starts, the selection), with the caret at sel within it. As in
// Chromium the marked text is part of the value, and an input event
// (Composing) follows every update. It reports whether an input was
// focused.
func (v *View) Compose(marked string, sel int) bool {
	n := v.focus
	st := v.input(n)
	if st == nil || n.Disabled() {
		return false
	}
	b := st.buf
	if !st.comp.active {
		if s, e := b.Selection(); s != e {
			b.Delete(false, edit.Char)
		}
		st.comp.active, st.comp.start, st.comp.n = true, b.Caret, 0
	}
	m := []rune(marked)
	st.comp.n = b.Replace(st.comp.start, st.comp.start+st.comp.n, string(m))
	b.SetCaret(st.comp.start+min(max(sel, 0), st.comp.n), false)
	v.reveal(n, st)
	v.dispatch(&Event{Type: "input", Target: n, Value: b.String(), Composing: true})
	return true
}

// Commit ends the composition with the final text (replacing the marked
// text; without a composition it inserts the text like typing).
func (v *View) Commit(s string) bool {
	n := v.focus
	st := v.input(n)
	if st == nil || n.Disabled() {
		return false
	}
	if !st.comp.active {
		return v.Text(s)
	}
	b := st.buf
	k := b.Replace(st.comp.start, st.comp.start+st.comp.n, s)
	b.SetCaret(st.comp.start+k, false)
	st.comp.active = false
	v.edited(n, st)
	return true
}

// CancelComposition removes the marked text and ends the composition.
func (v *View) CancelComposition() bool {
	n := v.focus
	st := v.input(n)
	if st == nil || !st.comp.active {
		return false
	}
	return v.Commit("")
}

// endComposition commits the marked text as it is (focus moves away or
// the pointer is pressed, as browsers do).
func (v *View) endComposition(n *dom.Node) {
	if st := v.input(n); st != nil && st.comp.active {
		st.comp.active = false
		v.needPaint = true
	}
}

// CaretRect returns the focused input's caret rectangle in document
// coordinates (for placing an IME candidate window).
func (v *View) CaretRect() (geom.Rect, bool) {
	n := v.focus
	st := v.input(n)
	if st == nil || v.doc == nil {
		return geom.Rect{}, false
	}
	return v.doc.InputCaretRect(n, st.buf.String(), st.buf.Caret, st.scrollX), true
}
