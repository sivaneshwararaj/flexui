// Package render turns HTML plus the CSS subset into pixels: it parses
// the document (dom), computes styles (style), lays out (layout), emits a
// display list in CSS paint order (paint.List) and rasterizes it
// (paint.Raster).
package render

import (
	"fmt"
	"image"
	"sort"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/layout"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Box is a rendered element: its computed style, layout node and
// absolute border box.
type Box struct {
	El    *dom.Node
	Style style.Style
	Lay   *layout.Node
	Kids  []*Box
	Abs   geom.Rect // border box in document coordinates, after layout
}

// Document is a styled tree ready for layout and painting.
type Document struct {
	Root     *Box
	Warnings []string
	seen     map[string]bool
}

func (d *Document) warn(format string, args ...any) {
	m := fmt.Sprintf(format, args...)
	if !d.seen[m] {
		d.seen[m] = true
		d.Warnings = append(d.Warnings, m)
	}
}

// Build parses html and styles it. css is applied first, followed by the
// document's own <style> elements in document order (spec/SUBSET.md §3).
func Build(html, css string) *Document {
	d := &Document{seen: map[string]bool{}}
	doc, warns := dom.Parse(html)
	for _, w := range warns {
		d.warn("%s", w)
	}
	var sheets []*style.Sheet
	order := 0
	for _, src := range append([]string{css}, doc.Styles...) {
		sh, ws := style.ParseSheet(src, order)
		order += len(sh.Rules) + 1
		for _, w := range ws {
			d.warn("css: %s", w.Error())
		}
		sheets = append(sheets, sh)
	}
	c := style.NewCascade(sheets...)
	d.Root = d.build(c, doc.Root, nil)
	return d
}

func (d *Document) build(c *style.Cascade, el *dom.Node, parent *style.Style) *Box {
	st, ws := c.Compute(el, parent, el.Attr("style"))
	for _, w := range ws {
		d.warn("style attribute: %s", w.Error())
	}
	b := &Box{El: el, Style: st}
	b.Lay = &layout.Node{Style: &b.Style}
	if el.Tag() == "img" {
		d.warn("<img> is not rendered yet: it is an empty box")
	}
	for _, k := range el.Children {
		if k.Kind == dom.TextNode {
			if !k.IsWhitespace() {
				d.warn("text is not rendered until milestone M4")
			}
			continue
		}
		kb := d.build(c, k, &b.Style)
		b.Kids = append(b.Kids, kb)
		b.Lay.Children = append(b.Lay.Children, kb.Lay)
	}
	return b
}

// Layout lays the document out in a viewport of w x h logical pixels.
func (d *Document) Layout(w, h int) {
	layout.Layout(d.Root.Lay, geom.Px(w), geom.Px(h))
	var place func(b *Box, ox, oy geom.Unit)
	place = func(b *Box, ox, oy geom.Unit) {
		bx := b.Lay.Box
		b.Abs = geom.Rect{X: ox + bx.X, Y: oy + bx.Y, W: bx.W, H: bx.H}
		for _, k := range b.Kids {
			place(k, b.Abs.X, b.Abs.Y)
		}
	}
	place(d.Root, 0, 0)
}

// Render renders html+css at w x h logical pixels and the given device
// scale into a new image, on an opaque white canvas.
func Render(html, css string, w, h int, scale float64) (*image.RGBA, []string) {
	img := image.NewRGBA(image.Rect(0, 0, int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)))
	warns := RenderInto(img, html, css, w, h, scale)
	return img, warns
}

// RenderInto is Render into an existing image (reused across frames).
func RenderInto(img *image.RGBA, html, css string, w, h int, scale float64) []string {
	d := Build(html, css)
	d.Layout(w, h)
	var l paint.List
	l.FillRRect(paint.Rect(geom.Rect{W: geom.Px(w), H: geom.Px(h)}), paint.Opaque(255, 255, 255))
	d.Paint(&l)
	var r paint.Raster
	r.Begin(img, paint.Scale64(scale))
	l.Replay(&r)
	r.End()
	return d.Warnings
}

// ---- Painting ----

func color(c style.Color) paint.Color { return paint.Color{R: c.R, G: c.G, B: c.B, A: c.A} }

// rrect returns the border box with resolved radii.
func (b *Box) rrect() paint.RRect {
	rr := paint.RRect{Rect: b.Abs}
	for i, r := range b.Style.BorderRadius {
		x, _ := r[0].Resolve(b.Abs.W, true)
		y, _ := r[1].Resolve(b.Abs.H, true)
		rr.Radii[i] = paint.Radius{X: max(x, 0), Y: max(y, 0)}
	}
	return rr
}

// clipRRect is the overflow clip: the padding box with inner radii.
func (b *Box) clipRRect() paint.RRect {
	bd := b.Lay.Box.Border
	rr := b.rrect()
	out := paint.RRect{Rect: geom.Rect{
		X: b.Abs.X + bd.Left, Y: b.Abs.Y + bd.Top,
		W: max(0, b.Abs.W-bd.Left-bd.Right), H: max(0, b.Abs.H-bd.Top-bd.Bottom),
	}}
	sub := func(r paint.Radius, h, v geom.Unit) paint.Radius {
		return paint.Radius{X: max(0, r.X-h), Y: max(0, r.Y-v)}
	}
	out.Radii[paint.TopLeft] = sub(rr.Radii[paint.TopLeft], bd.Left, bd.Top)
	out.Radii[paint.TopRight] = sub(rr.Radii[paint.TopRight], bd.Right, bd.Top)
	out.Radii[paint.BottomRight] = sub(rr.Radii[paint.BottomRight], bd.Right, bd.Bottom)
	out.Radii[paint.BottomLeft] = sub(rr.Radii[paint.BottomLeft], bd.Left, bd.Bottom)
	return out
}

func (b *Box) clips() bool { return b.Style.Overflow != style.OverflowVisible }

func (b *Box) hidden() bool { return b.Style.Display == style.DisplayNone }

// isContext: an element with z-index other than auto, or opacity < 1,
// establishes a stacking context (every element is positioned, D2).
func (b *Box) isContext() bool { return !b.Style.ZAuto || b.Style.Opacity < 1000 }

// z returns the stacking level of a context (auto counts as 0).
func (b *Box) z() int32 {
	if b.Style.ZAuto {
		return 0
	}
	return b.Style.ZIndex
}

// paintBox paints the element's own shadow, background and border.
func (b *Box) paintBox(c paint.Canvas) {
	st := &b.Style
	rr := b.rrect()
	if sh := st.Shadow; sh.Set && sh.Color.A > 0 {
		c.BoxShadow(rr, paint.Shadow{DX: sh.DX, DY: sh.DY, Blur: sh.Blur, Spread: sh.Spread, Color: color(sh.Color)})
	}
	if st.BackgroundColor.A > 0 {
		c.FillRRect(rr, color(st.BackgroundColor))
	}
	bd := b.Lay.Box.Border
	if (bd.Top|bd.Right|bd.Bottom|bd.Left) != 0 && st.BorderColor.A > 0 {
		c.Border(rr, bd, color(st.BorderColor))
	}
}

// deferred is a stacking context painted out of tree order, with the
// overflow clips of its ancestors (within the parent context).
type deferred struct {
	box   *Box
	clips []*Box
}

// Paint emits the document in paint order (spec/SUBSET.md §6.4).
//
// The root stacking context belongs to the canvas, not to <body>: body
// is an ordinary positioned element inside it (as in CSS, where the root
// context is <html>). So negative z-index boxes paint behind body's
// background, exactly as in browsers.
func (d *Document) Paint(c paint.Canvas) {
	canvas := &Box{Style: style.Initial(), Lay: &layout.Node{}, Kids: []*Box{d.Root}}
	canvas.context(c)
}

// context paints b as a stacking context (CSS 2 Appendix E, simplified
// because every element is positioned).
func (b *Box) context(c paint.Canvas) {
	if b.Style.Opacity < 1000 {
		c.PushOpacity(uint8((b.Style.Opacity*255 + 500) / 1000))
	}
	b.paintBox(c)
	var neg, pos []deferred
	var chain []*Box
	if b.clips() {
		chain = []*Box{b}
	}
	b.collect(chain, &neg, &pos)
	byZ := func(ds []deferred) {
		sort.SliceStable(ds, func(i, j int) bool { return ds[i].box.z() < ds[j].box.z() })
	}
	byZ(neg)
	byZ(pos)
	for _, d := range neg {
		d.paint(c)
	}
	if b.clips() {
		c.PushClip(b.clipRRect())
	}
	b.walk(c)
	if b.clips() {
		c.PopClip()
	}
	for _, d := range pos {
		d.paint(c)
	}
	if b.Style.Opacity < 1000 {
		c.PopOpacity()
	}
}

// collect finds the descendants of b (within b's stacking context) that
// are stacking contexts with a non-zero z-index.
func (b *Box) collect(chain []*Box, neg, pos *[]deferred) {
	for _, k := range b.Kids {
		if k.hidden() {
			continue
		}
		if k.isContext() {
			switch z := k.z(); {
			case z < 0:
				*neg = append(*neg, deferred{k, chain})
			case z > 0:
				*pos = append(*pos, deferred{k, chain})
			}
			continue // its descendants belong to its own context
		}
		next := chain
		if k.clips() {
			next = append(append([]*Box(nil), chain...), k)
		}
		k.collect(next, neg, pos)
	}
}

// walk paints b's descendants in tree order: boxes that are not stacking
// contexts, and z-index 0 contexts atomically.
func (b *Box) walk(c paint.Canvas) {
	for _, k := range b.Kids {
		if k.hidden() {
			continue
		}
		if k.isContext() {
			if k.z() == 0 {
				k.context(c)
			}
			continue
		}
		k.paintBox(c)
		if k.clips() {
			c.PushClip(k.clipRRect())
		}
		k.walk(c)
		if k.clips() {
			c.PopClip()
		}
	}
}

func (d deferred) paint(c paint.Canvas) {
	for _, cl := range d.clips {
		c.PushClip(cl.clipRRect())
	}
	d.box.context(c)
	for range d.clips {
		c.PopClip()
	}
}
