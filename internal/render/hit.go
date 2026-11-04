package render

import "github.com/sivaneshwararaj/flexui/internal/geom"

// HitTest returns the topmost element box at (x, y) (document
// coordinates, logical units): the last element in paint order whose
// border box contains the point and that is not clipped away there by an
// ancestor's overflow clip (spec/SUBSET.md §8.1). Text hits its parent
// element. Returns nil outside every element.
func (d *Document) HitTest(x, y geom.Unit) *Box {
	h := hitter{x: x, y: y}
	d.visit(&h)
	return h.hit
}

type hitter struct {
	x, y    geom.Unit
	clipped []bool // per pushed clip: does the point lie outside it?
	out     int    // number of clips the point is outside of
	hit     *Box
}

func contains(r geom.Rect, x, y geom.Unit) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (h *hitter) box(b *Box) {
	if b.El != nil && h.out == 0 && contains(b.Abs, h.x, h.y) {
		h.hit = b
	}
}

func (h *hitter) text(b *Box) {} // text belongs to its parent, which is already a candidate

func (h *hitter) pushClip(b *Box) {
	r := b.clipRRect().Rect
	o := !contains(r, h.x, h.y)
	h.clipped = append(h.clipped, o)
	if o {
		h.out++
	}
}

func (h *hitter) popClip() {
	if h.clipped[len(h.clipped)-1] {
		h.out--
	}
	h.clipped = h.clipped[:len(h.clipped)-1]
}

func (h *hitter) pushOpacity(*Box) {}
func (h *hitter) popOpacity()      {}

// Box returns the element box of node el, or nil.
func (d *Document) Box(match func(*Box) bool) *Box {
	var find func(b *Box) *Box
	find = func(b *Box) *Box {
		if b.El != nil && match(b) {
			return b
		}
		for _, k := range b.Kids {
			if r := find(k); r != nil {
				return r
			}
		}
		return nil
	}
	return find(d.Root)
}
