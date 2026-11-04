package render

import (
	_ "embed"
	"strconv"
	"strings"
	"sync"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/layout"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/style"
	"github.com/sivaneshwararaj/flexui/internal/text"
)

//go:embed ua.css
var uaCSS string

var (
	uaOnce sync.Once
	ua     *style.Sheet
)

// uaSheet returns the parsed default style sheet (spec/SUBSET.md §1.1).
func uaSheet() *style.Sheet {
	uaOnce.Do(func() {
		sh, w := style.ParseSheet(uaCSS, 0)
		if len(w) > 0 {
			panic("render: ua.css: " + w[0].Error())
		}
		ua = sh.MarkUA()
	})
	return ua
}

// isInline reports whether element el is an inline box: display: inline
// and only text and inline elements inside (otherwise it is treated as a
// flex item, as if display were flex).
func (d *Document) isInline(c *style.Cascade, el *dom.Node, parent *style.Style) bool {
	st := d.computed(c, el, parent)
	if st.Display != style.DisplayInline {
		return false
	}
	for _, k := range el.Children {
		if k.Kind == dom.ElementNode && !d.isInline(c, k, st) {
			if !d.seen["blockified"] {
				d.warn("<%s> contains <%s>, which is not inline; it is laid out as a flex item", el.Tag(), k.Tag())
			}
			return false
		}
	}
	return true
}

// outerOf copies the enclosing inline styles for a run (none for text
// directly in the block).
func outerOf(outer []text.Style, inInline bool) []text.Style {
	if !inInline || len(outer) == 0 {
		return nil
	}
	return append([]text.Style(nil), outer...)
}

// textStyle is the layout part of a computed style.
func textStyle(st *style.Style) text.Style {
	return text.Style{Weight: int(st.FontWeight), Size: st.FontSize, LineHeight: st.LineHeight}
}

// inlineBox makes the anonymous box for a group of text nodes and inline
// elements in parent (spec/SUBSET.md §7.1). It inherits the inherited
// properties; all others are initial, except that it takes `overflow` and
// `text-overflow` from a parent whose text-overflow is ellipsis and whose
// overflow clips (D24). It returns nil if the group has no text.
func (d *Document) inlineBox(c *style.Cascade, group []*dom.Node, parent *style.Style) *Box {
	st := style.Inherit(parent)
	if parent.TextOverflow == style.TextOverflowEllipsis && parent.Overflow != style.OverflowVisible {
		st.Overflow, st.TextOverflow = parent.Overflow, parent.TextOverflow
	}
	st.Finish()
	b := &Box{Style: st}
	var runs []text.Run
	var outer []text.Style // styles of the enclosing inline boxes
	var collect func(nodes []*dom.Node, owner *Box, ost *style.Style)
	collect = func(nodes []*dom.Node, owner *Box, ost *style.Style) {
		for _, n := range nodes {
			if n.Kind == dom.TextNode {
				runs = append(runs, text.Run{Text: n.Text, Style: textStyle(ost), Outer: outerOf(outer, owner != nil)})
				b.runOwner = append(b.runOwner, owner)
				continue
			}
			est := d.computed(c, n, ost)
			ib := &Box{El: n, Style: *est, Inline: true, anon: b, first: len(runs)}
			d.boxes[n] = ib
			if est.Display == style.DisplayNone {
				continue
			}
			d.checkInline(ib)
			b.inlines = append(b.inlines, ib)
			if n.Tag() == "br" {
				runs = append(runs, text.Run{Break: true, Style: textStyle(est), Outer: outerOf(append(outer, textStyle(ost)), owner != nil)})
				b.runOwner = append(b.runOwner, ib)
			} else {
				if owner != nil {
					outer = append(outer, textStyle(ost))
				}
				collect(n.Children, ib, &ib.Style)
				if owner != nil {
					outer = outer[:len(outer)-1]
				}
			}
			ib.last = len(runs)
		}
	}
	collect(group, nil, &b.Style)
	p := text.Params{
		Style:      textStyle(&st),
		WhiteSpace: st.WhiteSpace, Align: st.TextAlign,
		Ellipsis: st.TextOverflow == style.TextOverflowEllipsis,
	}
	t := d.cache.text(runs, p)
	if t.Empty() {
		for _, ib := range b.inlines {
			delete(d.boxes, ib.El)
		}
		return nil
	}
	b.Text = t
	b.Lay = &layout.Node{Style: &b.Style, Measure: textMeasure{t}}
	return b
}

// checkInline warns about properties that have no effect on inline boxes.
func (d *Document) checkInline(b *Box) {
	st, in := &b.Style, style.Initial()
	var bad []string
	if st.Padding != in.Padding {
		bad = append(bad, "padding")
	}
	if st.Margin != in.Margin {
		bad = append(bad, "margin")
	}
	if st.Border != in.Border {
		bad = append(bad, "border")
	}
	if st.Width != in.Width || st.Height != in.Height {
		bad = append(bad, "width/height")
	}
	if st.Position != in.Position {
		bad = append(bad, "position")
	}
	if st.Opacity != in.Opacity || st.Shadow.Set || st.BorderRadius != in.BorderRadius {
		bad = append(bad, "opacity/box-shadow/border-radius")
	}
	if st.Overflow != in.Overflow {
		bad = append(bad, "overflow")
	}
	for _, p := range bad {
		d.warn("%s has no effect on inline elements (<%s>)", p, b.El.Tag())
	}
}

// placeInline computes the fragments of the inline boxes of anonymous box
// b after b.Abs is known: per line, the horizontal extent of the box's
// text and the content area of its own font (CSS 2.1 §10.6.1).
func (b *Box) placeInline() {
	if len(b.inlines) == 0 {
		return
	}
	t := b.Text
	blk := t.Layout(b.Abs.W)
	for _, ib := range b.inlines {
		ib.Frags = ib.Frags[:0]
		s, e := t.RuneRange(ib.first, ib.last)
		asc, desc := text.Metrics(textStyle(&ib.Style))
		var u geom.Rect
		for k := range blk.Lines {
			ln := &blk.Lines[k]
			fs, fe := max(s, ln.Start), min(e, ln.FullEnd)
			if fs >= fe && !(s == e && s >= ln.Start && s <= ln.FullEnd && ib.El.Tag() != "br") {
				continue
			}
			x0, x1 := t.X(ln, fs), t.X(ln, max(fs, fe))
			r := geom.Rect{X: b.Abs.X + x0, Y: b.Abs.Y + ln.Baseline - asc, W: x1 - x0, H: asc + desc}
			ib.Frags = append(ib.Frags, r)
			u = union(u, r, len(ib.Frags) == 1)
		}
		ib.Abs = u
	}
}

func union(u, r geom.Rect, first bool) geom.Rect {
	if first {
		return r
	}
	x0, y0 := min(u.X, r.X), min(u.Y, r.Y)
	x1, y1 := max(u.X+u.W, r.X+r.W), max(u.Y+u.H, r.Y+r.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// paintText draws an anonymous box: inline backgrounds, then the glyphs
// of each run in its owner's color.
func (b *Box) paintText(c paint.Canvas) {
	t := b.Text
	if b.clips() {
		c.PushClip(paint.Rect(b.Abs))
	}
	for _, ib := range b.inlines {
		if ib.Style.BackgroundColor.A == 0 {
			continue
		}
		for _, f := range ib.Frags {
			c.FillRRect(paint.Rect(f), color(ib.Style.BackgroundColor))
		}
	}
	blk := t.Layout(b.Abs.W)
	var gs []text.PlacedGlyph
	var run *paint.GlyphRun
	cur := -2
	flush := func() {
		if run != nil && len(run.Glyphs) > 0 && run.Color.A > 0 {
			c.Glyphs(run)
		}
		run = nil
	}
	for i := range blk.Lines {
		gs = t.Glyphs(&blk.Lines[i], gs[:0])
		for _, g := range gs {
			if g.Run != cur || run == nil {
				flush()
				cur = g.Run
				var f *text.Face
				var size geom.Unit
				col := b.Style.Color
				if g.Run < 0 {
					f, size = t.StrutFace()
				} else {
					f, size = t.RunFace(g.Run)
					if o := b.runOwner[g.Run]; o != nil {
						col = o.Style.Color
					}
				}
				run = &paint.GlyphRun{Font: f, Size: size, Color: color(col)}
			}
			run.Glyphs = append(run.Glyphs, paint.Glyph{ID: g.ID, X: b.Abs.X + g.X, Y: b.Abs.Y + g.Y})
		}
		cur = -2 // new line: new run (keeps runs short and ordered)
	}
	flush()
	if b.clips() {
		c.PopClip()
	}
}

// textKey encodes runs and parameters for the text cache.
func textKey(runs []text.Run, p text.Params) string {
	var k strings.Builder
	st := func(s text.Style) {
		k.WriteString(strconv.Itoa(s.Weight))
		k.WriteByte(',')
		k.WriteString(strconv.FormatInt(int64(s.Size), 10))
		k.WriteByte(',')
		k.WriteString(strconv.Itoa(int(s.LineHeight.Kind)))
		k.WriteByte(',')
		k.WriteString(strconv.FormatInt(int64(s.LineHeight.V), 10))
	}
	st(p.Style)
	k.WriteString("|" + strconv.Itoa(int(p.WhiteSpace)) + strconv.Itoa(int(p.Align)) + strconv.FormatBool(p.Ellipsis))
	for _, r := range runs {
		k.WriteByte('|')
		st(r.Style)
		for _, o := range r.Outer {
			k.WriteByte('<')
			st(o)
		}
		if r.Break {
			k.WriteString(",br")
			continue
		}
		k.WriteByte(':')
		k.WriteString(strconv.Itoa(len(r.Text)))
		k.WriteByte(':')
		k.WriteString(r.Text)
	}
	return k.String()
}
