package view

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Event is one DOM-like event being dispatched.
type Event struct {
	Type    string    // "click", ...
	Target  *dom.Node // the deepest element the event is for
	Current *dom.Node // the element whose handler is being called
	Name    string    // handler name from the on<type> attribute
	X, Y    geom.Unit // pointer position (pointer events)
	Key     string    // key name (key events)
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

// PointerDown handles a primary-button press at (x, y).
func (v *View) PointerDown(x, y geom.Unit) {
	v.down = v.hit(x, y)
	v.downAt = geom.Point{X: x, Y: y}
	v.pressed = true
}

// PointerUp handles the release; a press and release form a click on
// their nearest common ancestor element (as in browsers). It reports
// whether a handler ran.
func (v *View) PointerUp(x, y geom.Unit) bool {
	if !v.pressed {
		return false
	}
	v.pressed = false
	up := v.hit(x, y)
	t := commonAncestor(v.down, up)
	v.down = nil
	if t == nil {
		return false
	}
	return v.dispatch(&Event{Type: "click", Target: t, X: x, Y: y})
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
