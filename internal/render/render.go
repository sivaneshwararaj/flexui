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
	"github.com/sivaneshwararaj/flexui/internal/text"
)

// Box is a rendered element: its computed style, layout node and
// absolute border box.
type Box struct {
	El    *dom.Node // nil for an anonymous text box
	Style style.Style
	Lay   *layout.Node
	Kids  []*Box
	Abs   geom.Rect  // border box in document coordinates, after layout
	Text  *text.Text // set for anonymous text boxes
}

// textMeasure adapts a text run to layout.Measurer.
type textMeasure struct{ t *text.Text }

func (m textMeasure) Widths() (geom.Unit, geom.Unit) { return m.t.MinContent(), m.t.MaxContent() }
func (m textMeasure) Height(w geom.Unit) geom.Unit   { return m.t.Height(w) }

// newText makes the anonymous box for a run of text in parent (spec
// SUBSET.md §7.1). It inherits the inherited properties; all others are
// initial, except that it takes `overflow` and `text-overflow` from a
// parent whose text-overflow is ellipsis and whose overflow clips.
func (c *Cache) newText(raw string, parent *style.Style) *Box {
	st := style.Inherit(parent)
	if parent.TextOverflow == style.TextOverflowEllipsis && parent.Overflow != style.OverflowVisible {
		st.Overflow, st.TextOverflow = parent.Overflow, parent.TextOverflow
	}
	st.Finish()
	t := c.text(raw, text.Params{
		Weight: int(st.FontWeight), Size: st.FontSize, LineHeight: st.LineHeight,
		WhiteSpace: st.WhiteSpace, Align: st.TextAlign,
		Ellipsis: st.TextOverflow == style.TextOverflowEllipsis,
	})
	if t.Empty() {
		return nil
	}
	b := &Box{Style: st, Text: t}
	b.Lay = &layout.Node{Style: &b.Style, Measure: textMeasure{t}}
	return b
}

// Document is a styled tree ready for layout and painting.
type Document struct {
	Dom      *dom.Document
	Root     *Box
	Warnings []string
	seen     map[string]bool
	cache    *Cache
}

func (d *Document) warn(format string, args ...any) {
	m := fmt.Sprintf(format, args...)
	if !d.seen[m] {
		d.seen[m] = true
		d.Warnings = append(d.Warnings, m)
	}
}

// Parsed is a parsed HTML document with its parse warnings. It can be
// styled any number of times (e.g. after a :hover change).
type Parsed struct {
	Dom      *dom.Document
	Warnings []string
}

// Parse parses an HTML document.
func Parse(html string) *Parsed {
	doc, warns := dom.Parse(html)
	return &Parsed{doc, warns}
}

// Build parses html and styles it. css is applied first, followed by the
// document's own <style> elements in document order (spec/SUBSET.md §3).
func Build(html, css string) *Document {
	return NewCache().Style(Parse(html), css)
}

// Style computes styles for a parsed document (using the dom nodes'
// current pseudo-class state) and builds the box tree.
func (c *Cache) Style(p *Parsed, css string) *Document {
	c.Stats.Styles++
	c.gen++
	d := &Document{Dom: p.Dom, seen: map[string]bool{}, cache: c}
	for _, w := range p.Warnings {
		d.warn("%s", w)
	}
	var sheets []*style.Sheet
	order := 0
	for _, src := range append([]string{css}, p.Dom.Styles...) {
		sh, ws := c.sheet(src, order)
		order += len(sh.Rules) + 1
		for _, w := range ws {
			d.warn("css: %s", w.Error())
		}
		sheets = append(sheets, sh)
	}
	d.Root = d.build(style.NewCascade(sheets...), p.Dom.Root, nil)
	c.sweep()
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
	kids := el.Children
	for i := 0; i < len(kids); i++ {
		var kb *Box
		if kids[i].Kind == dom.TextNode {
			// Adjacent text nodes form one anonymous run.
			raw := kids[i].Text
			for i+1 < len(kids) && kids[i+1].Kind == dom.TextNode {
				i++
				raw += kids[i].Text
			}
			if kb = d.cache.newText(raw, &b.Style); kb == nil {
				continue
			}
		} else {
			kb = d.build(c, kids[i], &b.Style)
		}
		b.Kids = append(b.Kids, kb)
		b.Lay.Children = append(b.Lay.Children, kb.Lay)
	}
	return b
}

// Layout lays the document out in a viewport of w x h logical pixels and
// places the boxes (Place with no scrolling).
func (d *Document) Layout(w, h int) {
	if d.cache != nil {
		d.cache.Stats.Layouts++
	}
	layout.Layout(d.Root.Lay, geom.Px(w), geom.Px(h))
	d.Place(nil)
}

// Place computes absolute border boxes from the layout result. scroll, if
// not nil, returns the scroll offset of a scroll container's content; its
// descendants are moved up and left by it.
func (d *Document) Place(scroll func(*Box) geom.Point) {
	var place func(b *Box, ox, oy geom.Unit)
	place = func(b *Box, ox, oy geom.Unit) {
		bx := b.Lay.Box
		b.Abs = geom.Rect{X: ox + bx.X, Y: oy + bx.Y, W: bx.W, H: bx.H}
		cx, cy := b.Abs.X, b.Abs.Y
		if scroll != nil && b.El != nil {
			o := scroll(b)
			cx, cy = cx-o.X, cy-o.Y
		}
		for _, k := range b.Kids {
			place(k, cx, cy)
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

// RenderInto is Render into an existing image.
func RenderInto(img *image.RGBA, html, css string, w, h int, scale float64) []string {
	d := Build(html, css)
	d.Layout(w, h)
	var l paint.List
	var r paint.Raster
	d.Draw(img, w, h, scale, &l, &r)
	return d.Warnings
}

// Draw paints the laid-out document into img on an opaque white canvas,
// reusing the list and raster buffers.
func (d *Document) Draw(img *image.RGBA, w, h int, scale float64, l *paint.List, r *paint.Raster) {
	if d.cache != nil {
		d.cache.Stats.Paints++
	}
	l.Reset()
	l.FillRRect(paint.Rect(geom.Rect{W: geom.Px(w), H: geom.Px(h)}), paint.Opaque(255, 255, 255))
	d.Paint(l)
	r.Begin(img, paint.Scale64(scale))
	l.Replay(r)
	r.End()
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

// visitor receives the document in paint order. Painting and hit testing
// share the traversal, so hit testing follows exactly the paint order.
type visitor interface {
	box(b *Box)  // an element's own shadow, background and border
	text(b *Box) // an anonymous text box
	pushClip(b *Box)
	popClip()
	pushOpacity(b *Box)
	popOpacity()
}

// Paint emits the document in paint order (spec/SUBSET.md §6.4).
//
// The root stacking context belongs to the canvas, not to <body>: body
// is an ordinary positioned element inside it (as in CSS, where the root
// context is <html>). So negative z-index boxes paint behind body's
// background, exactly as in browsers.
func (d *Document) Paint(c paint.Canvas) { d.visit(painter{c}) }

func (d *Document) visit(v visitor) {
	canvas := &Box{Style: style.Initial(), Lay: &layout.Node{}, Kids: []*Box{d.Root}}
	canvas.context(v)
}

// context visits b as a stacking context (CSS 2 Appendix E, simplified
// because every element is positioned).
func (b *Box) context(v visitor) {
	if b.Style.Opacity < 1000 {
		v.pushOpacity(b)
	}
	v.box(b)
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
		d.visit(v)
	}
	if b.clips() {
		v.pushClip(b)
	}
	b.walk(v)
	if b.clips() {
		v.popClip()
	}
	for _, d := range pos {
		d.visit(v)
	}
	if b.Style.Opacity < 1000 {
		v.popOpacity()
	}
}

// collect finds the descendants of b (within b's stacking context) that
// are stacking contexts with a non-zero z-index.
func (b *Box) collect(chain []*Box, neg, pos *[]deferred) {
	for _, k := range b.Kids {
		if k.hidden() || k.Text != nil {
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

// walk visits b's descendants in tree order: boxes that are not stacking
// contexts, and z-index 0 contexts atomically.
func (b *Box) walk(v visitor) {
	// Anonymous text boxes are not positioned: they paint with their
	// parent, before its (positioned) element children (§6.4).
	for _, k := range b.Kids {
		if k.Text != nil {
			v.text(k)
		}
	}
	for _, k := range b.Kids {
		if k.hidden() || k.Text != nil {
			continue
		}
		if k.isContext() {
			if k.z() == 0 {
				k.context(v)
			}
			continue
		}
		v.box(k)
		if k.clips() {
			v.pushClip(k)
		}
		k.walk(v)
		if k.clips() {
			v.popClip()
		}
	}
}

func (d deferred) visit(v visitor) {
	for _, cl := range d.clips {
		v.pushClip(cl)
	}
	d.box.context(v)
	for range d.clips {
		v.popClip()
	}
}

// painter is the visitor that draws.
type painter struct{ c paint.Canvas }

func (p painter) box(b *Box) {
	if b.El != nil {
		b.paintBox(p.c)
	}
}
func (p painter) text(b *Box)        { b.paintText(p.c) }
func (p painter) pushClip(b *Box)    { p.c.PushClip(b.clipRRect()) }
func (p painter) popClip()           { p.c.PopClip() }
func (p painter) pushOpacity(b *Box) { p.c.PushOpacity(uint8((b.Style.Opacity*255 + 500) / 1000)) }
func (p painter) popOpacity()        { p.c.PopOpacity() }

// paintText draws an anonymous text box's lines.
func (b *Box) paintText(c paint.Canvas) {
	t := b.Text
	if b.Style.Color.A == 0 {
		return
	}
	if b.clips() {
		c.PushClip(paint.Rect(b.Abs))
	}
	blk := t.Layout(b.Abs.W)
	run := &paint.GlyphRun{Font: t.Face(), Size: t.Params().Size, Color: color(b.Style.Color)}
	var gs []text.PlacedGlyph
	for i := range blk.Lines {
		gs = t.Glyphs(&blk.Lines[i], gs[:0])
		for _, g := range gs {
			run.Glyphs = append(run.Glyphs, paint.Glyph{ID: g.ID, X: b.Abs.X + g.X, Y: b.Abs.Y + g.Y})
		}
	}
	c.Glyphs(run)
	if b.clips() {
		c.PopClip()
	}
}
