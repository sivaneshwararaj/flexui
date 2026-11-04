package render

import (
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Scrollbars (spec/SUBSET.md §8.5): overlay scrollbars, drawn over the
// content of every overflow: scroll box whose content overflows, inside
// its padding box. They take no layout space (D25). Only the thumb is
// drawn: 6px wide, 2px from the edges, at least 20px long, rounded,
// black at 35% opacity.

var scrollbarColor = paint.Color{A: 89}

const (
	barWidth  = 6 * geom.UnitsPerPx
	barInset  = 2 * geom.UnitsPerPx
	minThumb  = 20 * geom.UnitsPerPx
	barMargin = barWidth + 2*barInset // space kept free for the other bar
)

// Thumb is a scrollbar thumb: its rectangle and the track it moves in
// (document coordinates), and its axis (0 horizontal, 1 vertical).
type Thumb struct {
	Rect  geom.Rect
	Track geom.Rect
	Axis  int
}

// Client returns the padding box of b (document coordinates).
func (b *Box) Client() geom.Rect {
	bd := b.Lay.Box.Border
	return geom.Rect{X: b.Abs.X + bd.Left, Y: b.Abs.Y + bd.Top,
		W: max(0, b.Abs.W-bd.Left-bd.Right), H: max(0, b.Abs.H-bd.Top-bd.Bottom)}
}

// MaxScroll returns the scroll range of b (zero if it does not overflow).
func (b *Box) MaxScroll() geom.Point {
	if b.Lay == nil {
		return geom.Point{}
	}
	c := b.Client()
	return geom.Point{X: max(0, b.Lay.Box.ContentW-c.W), Y: max(0, b.Lay.Box.ContentH-c.H)}
}

// Thumbs returns the scrollbar thumbs of a scroll container (none for
// other boxes or when nothing overflows).
func (b *Box) Thumbs() []Thumb {
	if b.Inline || b.Lay == nil || b.Style.Overflow != style.OverflowScroll {
		return nil
	}
	m := b.MaxScroll()
	c := b.Client()
	var out []Thumb
	both := m.X > 0 && m.Y > 0
	if m.Y > 0 {
		tr := geom.Rect{X: c.X + c.W - barInset - barWidth, Y: c.Y + barInset, W: barWidth, H: c.H - 2*barInset}
		if both {
			tr.H -= barMargin - barInset
		}
		out = append(out, thumb(tr, 1, c.H, c.H+m.Y, b.Scroll.Y, m.Y))
	}
	if m.X > 0 {
		tr := geom.Rect{X: c.X + barInset, Y: c.Y + c.H - barInset - barWidth, W: c.W - 2*barInset, H: barWidth}
		if both {
			tr.W -= barMargin - barInset
		}
		out = append(out, thumb(tr, 0, c.W, c.W+m.X, b.Scroll.X, m.X))
	}
	return out
}

func thumb(track geom.Rect, axis int, client, content, off, maxOff geom.Unit) Thumb {
	tl := track.W
	if axis == 1 {
		tl = track.H
	}
	n := max(min(geom.Unit(geom.MulDiv(int64(tl), int64(client), int64(content))), tl), min(minThumb, tl))
	pos := geom.Unit(0)
	if maxOff > 0 {
		pos = geom.Unit(geom.MulDiv(int64(tl-n), int64(off), int64(maxOff)))
	}
	t := Thumb{Track: track, Axis: axis, Rect: track}
	if axis == 1 {
		t.Rect.Y, t.Rect.H = track.Y+pos, n
	} else {
		t.Rect.X, t.Rect.W = track.X+pos, n
	}
	return t
}
