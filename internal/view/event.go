package view

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Event is one DOM-like event being dispatched.
type Event struct {
	Type    string    // "click", "keydown"
	Target  *dom.Node // the deepest element the event is for
	Current *dom.Node // the element whose handler is being called
	Name    string    // handler name from the on<type> attribute
	X, Y    geom.Unit // pointer position (pointer events)
	Key     string    // key name (key events)
	Value   string    // the input's text (input, change, and key events on inputs)
	// Composing is set on input events during an IME composition.
	Composing bool
	Mods      uint8 // modifier keys (key events)
	// Prevented is set by a handler to suppress the default action (for
	// keydown: focus traversal and activation).
	Prevented bool
}

// dispatch calls the handlers for ev along the propagation path: the
// target, then its ancestors (bubbling), until a handler stops it.
// It reports whether any handler ran.
func (v *View) dispatch(ev *Event) bool {
	ran := false
	attr := "on" + ev.Type
	for n := ev.Target; n != nil; n = n.Parent {
		name := n.Attr(attr)
		if name == "" {
			continue
		}
		ev.Current, ev.Name = n, name
		ran = true
		if v.Handler != nil && v.Handler(ev) {
			break
		}
	}
	return ran
}

// dispatchSelf calls the handler of ev's target only (focus, blur).
func (v *View) dispatchSelf(ev *Event) bool {
	name := ev.Target.Attr("on" + ev.Type)
	if name == "" || v.Handler == nil {
		return false
	}
	ev.Current, ev.Name = ev.Target, name
	v.Handler(ev)
	return true
}

// disabledPath reports whether n or an ancestor is a disabled control:
// such elements receive no clicks, and neither do their ancestors through
// them (as in browsers).
func disabledPath(n *dom.Node) bool {
	for ; n != nil; n = n.Parent {
		if n.Disabled() {
			return true
		}
	}
	return false
}

// dragSlop is how far (logical px) the pointer must move while pressed
// before a press over a scroll container becomes a drag scroll.
const dragSlop = 4 * geom.UnitsPerPx

// PointerMove handles pointer movement: it updates :hover, and while the
// button is pressed it may start or continue drag scrolling.
func (v *View) PointerMove(x, y geom.Unit) {
	v.pointer, v.hasPointer = geom.Point{X: x, Y: y}, true
	if v.Document() == nil {
		return
	}
	if v.pressed && v.thumb != nil {
		v.dragThumb(x, y)
	} else if v.pressed && v.selecting != nil {
		if st := v.input(v.selecting); st != nil {
			st.buf.SetCaret(v.indexAt(v.selecting, st, x), true)
			v.reveal(v.selecting, st)
		}
	} else if v.pressed {
		if v.dragging == nil && (abs(x-v.downAt.X) > dragSlop || abs(y-v.downAt.Y) > dragSlop) {
			if sc := v.scrollerFor(v.down); sc != nil {
				v.dragging, v.dragLast = sc, v.downAt
				v.setActive(nil) // a drag is not a click
			}
		}
		if v.dragging != nil {
			v.scrollBy(v.dragging, v.dragLast.X-x, v.dragLast.Y-y, false)
			v.dragLast = geom.Point{X: x, Y: y}
		}
	}
	v.hoverStale = true
	v.update(v.w, v.h)
}

// PointerDown handles a primary-button press at (x, y): :active, focus,
// and the start of a click or drag.
func (v *View) PointerDown(x, y geom.Unit) {
	v.endComposition(v.focus) // a press commits the marked text, as in browsers
	v.pointer, v.hasPointer = geom.Point{X: x, Y: y}, true
	v.down = v.hit(x, y)
	v.downAt = geom.Point{X: x, Y: y}
	v.pressed = true
	v.dragging = nil
	v.thumb = v.thumbAt(v.down, x, y)
	if v.thumb != nil {
		return // dragging a scrollbar thumb: no :active, focus or click
	}
	if disabledPath(v.down) {
		v.setActive(nil)
	} else {
		v.setActive(v.down)
	}
	// Focus the nearest focusable element; pressing elsewhere blurs.
	var f *dom.Node
	for n := v.down; n != nil; n = n.Parent {
		if v.focusable(n) {
			f = n
			break
		}
	}
	v.setFocus(f)
	if st := v.input(f); st != nil && f == v.down {
		st.buf.SetCaret(v.indexAt(f, st, x), false)
		v.selecting = f
		v.reveal(f, st)
	}
}

// PointerUp handles the release; a press and release form a click on
// their nearest common ancestor element (as in browsers), unless the
// press turned into a drag scroll. It reports whether a handler ran.
func (v *View) PointerUp(x, y geom.Unit) bool {
	v.pointer, v.hasPointer = geom.Point{X: x, Y: y}, true
	if !v.pressed {
		return false
	}
	v.pressed = false
	v.selecting = nil
	v.setActive(nil)
	if v.thumb != nil {
		v.thumb, v.down = nil, nil
		return false
	}
	if v.dragging != nil {
		v.dragging, v.down = nil, nil
		return false
	}
	up := v.hit(x, y)
	t := commonAncestor(v.down, up)
	v.down = nil
	if t == nil || disabledPath(t) {
		return false
	}
	return v.dispatch(&Event{Type: "click", Target: t, X: x, Y: y})
}

// PointerLeave handles the pointer leaving the surface: no element is
// hovered.
func (v *View) PointerLeave() {
	v.hasPointer = false
	if v.doc != nil {
		v.setHover(nil)
	}
}

// Wheel scrolls the scroll container under (x, y) by (dx, dy) logical
// units; what it cannot absorb goes to the next scroll container outward
// (scroll chaining). It reports whether anything scrolled.
func (v *View) Wheel(x, y, dx, dy geom.Unit) bool {
	v.pointer, v.hasPointer = geom.Point{X: x, Y: y}, true
	n := v.hit(x, y)
	moved := false
	for ; n != nil && (dx != 0 || dy != 0); n = n.Parent {
		if b := v.doc.BoxOf(n); b != nil && scrollable(b) {
			var m bool
			dx, dy, m = v.scrollBy(n, dx, dy, true)
			moved = moved || m
		}
	}
	return moved
}

// KeyDown handles a key press: keydown handlers on the focused element
// (or body) and their ancestors, then, unless prevented, the default
// action: Tab / Shift+Tab move focus, Enter activates a focused button or
// link, Space presses a focused button (activated on release).
func (v *View) KeyDown(k string, mods uint8, shift bool) bool {
	d := v.Document()
	if d == nil {
		return false
	}
	t := v.focus
	if t == nil {
		t = d.Dom.Root
	}
	ev := &Event{Type: "keydown", Target: t, Key: k, Mods: mods}
	if st := v.input(v.focus); st != nil {
		ev.Value = st.buf.String()
	}
	ran := v.dispatch(ev)
	if ev.Prevented {
		return ran
	}
	if st := v.input(v.focus); st != nil && !v.focus.Disabled() && k != "Tab" {
		if v.editKey(v.focus, st, k, mods) {
			return true
		}
	}
	switch k {
	case "Tab":
		v.moveFocus(shift)
	case "Enter":
		if f := v.focus; f != nil && (f.Tag() == "button" || f.Tag() == "a") {
			ran = v.dispatch(&Event{Type: "click", Target: f}) || ran
		}
	case " ":
		if f := v.focus; f != nil && f.Tag() == "button" {
			v.space = f
			v.setActive(f)
		}
	}
	return ran
}

// KeyUp handles a key release (Space activates the button it pressed).
func (v *View) KeyUp(k string) bool {
	if k == " " && v.space != nil {
		f := v.space
		v.space = nil
		v.setActive(nil)
		if f == v.focus {
			return v.dispatch(&Event{Type: "click", Target: f})
		}
	}
	return false
}

func commonAncestor(a, b *dom.Node) *dom.Node {
	for x := a; x != nil; x = x.Parent {
		for y := b; y != nil; y = y.Parent {
			if x == y {
				return x
			}
		}
	}
	return nil
}

func abs(u geom.Unit) geom.Unit {
	if u < 0 {
		return -u
	}
	return u
}
