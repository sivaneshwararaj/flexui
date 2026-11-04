package view

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// thumbDrag is a scrollbar thumb being dragged.
type thumbDrag struct {
	el     *dom.Node
	axis   int
	start  geom.Point // scroll offset at press
	at     geom.Unit  // pointer coordinate along the axis at press
	travel geom.Unit  // track length minus thumb length
	max    geom.Unit  // scroll range along the axis
}

// thumbAt returns a drag for the scrollbar thumb under (x, y) of the
// nearest scroll container at or above n, or nil.
func (v *View) thumbAt(n *dom.Node, x, y geom.Unit) *thumbDrag {
	for ; n != nil; n = n.Parent {
		b := v.doc.BoxOf(n)
		if b == nil || !scrollable(b) {
			continue
		}
		for _, t := range b.Thumbs() {
			r := t.Rect
			if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
				d := &thumbDrag{el: n, axis: t.Axis, start: b.Scroll}
				m := b.MaxScroll()
				if t.Axis == 1 {
					d.at, d.travel, d.max = y, t.Track.H-r.H, m.Y
				} else {
					d.at, d.travel, d.max = x, t.Track.W-r.W, m.X
				}
				return d
			}
		}
		return nil // only the innermost scroll container's bars
	}
	return nil
}

// dragThumb moves the content so the thumb follows the pointer.
func (v *View) dragThumb(x, y geom.Unit) {
	d := v.thumb
	if d.travel <= 0 {
		return
	}
	p := x
	if d.axis == 1 {
		p = y
	}
	delta := geom.Unit(geom.MulDiv(int64(p-d.at), int64(d.max), int64(d.travel)))
	b := v.doc.BoxOf(d.el)
	if b == nil {
		return
	}
	cur := b.Scroll
	want := d.start
	if d.axis == 1 {
		want.Y += delta
	} else {
		want.X += delta
	}
	v.scrollBy(d.el, want.X-cur.X, want.Y-cur.Y, false)
}
