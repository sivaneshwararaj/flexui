package render

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/style"
	"github.com/sivaneshwararaj/flexui/internal/text"
)

// inputMeasure sizes an <input> (spec/SUBSET.md §1.3): its content is one
// line of text in its font, and its intrinsic width is 160px (the width
// of the default style sheet), so a flex row does not shrink it below
// that (as in Chromium).
type inputMeasure struct{ h geom.Unit }

const inputWidth = 160 * geom.UnitsPerPx

func (m inputMeasure) Widths() (geom.Unit, geom.Unit) { return inputWidth, inputWidth }
func (m inputMeasure) Height(geom.Unit) geom.Unit     { return m.h }

func (d *Document) input(b *Box) {
	b.Lay.Measure = inputMeasure{text.LineHeightOf(textStyle(&b.Style))}
}

// InputState is what is drawn in an <input> (spec/SUBSET.md §8.6).
type InputState struct {
	Text          string
	Caret, Anchor int       // rune indices; selection between them
	Focused       bool      // draw the caret and selection
	ScrollX       geom.Unit // horizontal scroll of the text
}

var (
	selectionColor   = paint.Color{R: 179, G: 213, B: 255, A: 255}
	placeholderColor = paint.Color{R: 154, G: 160, B: 168, A: 255}
)

// contentBox returns b's content box (document coordinates).
func (b *Box) contentBox() geom.Rect {
	bx := b.Lay.Box
	return geom.Rect{
		X: b.Abs.X + bx.Border.Left + bx.Padding.Left,
		Y: b.Abs.Y + bx.Border.Top + bx.Padding.Top,
		W: max(0, b.Abs.W-bx.Border.Left-bx.Border.Right-bx.Padding.Left-bx.Padding.Right),
		H: max(0, b.Abs.H-bx.Border.Top-bx.Border.Bottom-bx.Padding.Top-bx.Padding.Bottom),
	}
}

// inputText returns the laid-out single line of s in the input's style.
func (d *Document) inputText(b *Box, s string) (*text.Text, *text.Line) {
	p := text.Params{Style: textStyle(&b.Style), WhiteSpace: style.WhiteSpacePre}
	t := d.cache.text([]text.Run{{Text: s, Style: p.Style}}, p)
	blk := t.Layout(1 << 40)
	if len(blk.Lines) == 0 {
		return t, &text.Line{Baseline: t.Ascent(), Height: t.LineHeight()}
	}
	return t, &blk.Lines[0]
}

// InputCaretX returns the x offset of rune i in the input's text (relative
// to its content box, unscrolled), and the width of the content box.
func (d *Document) InputCaretX(n *dom.Node, s string, i int) (x, width geom.Unit) {
	b := d.boxes[n]
	if b == nil || b.Lay == nil {
		return 0, 0
	}
	t, ln := d.inputText(b, s)
	if len([]rune(s)) == 0 {
		return 0, b.contentBox().W
	}
	return t.X(ln, i), b.contentBox().W
}

// InputTextWidth returns the advance width of s in the input's font.
func (d *Document) InputTextWidth(n *dom.Node, s string) geom.Unit {
	b := d.boxes[n]
	if b == nil || b.Lay == nil || s == "" {
		return 0
	}
	_, ln := d.inputText(b, s)
	return ln.Width
}

// InputContent returns the content box of input n (document coordinates).
func (d *Document) InputContent(n *dom.Node) geom.Rect {
	if b := d.boxes[n]; b != nil && b.Lay != nil {
		return b.contentBox()
	}
	return geom.Rect{}
}

// paintInput draws an input's text (or placeholder), selection and caret,
// clipped to its content box.
func (d *Document) paintInput(c paint.Canvas, b *Box, st InputState) {
	cb := b.contentBox()
	c.PushClip(paint.Rect(cb))
	defer c.PopClip()
	s, col := st.Text, b.Style.Color
	ph := s == ""
	if ph {
		s = b.El.Attr("placeholder")
	}
	t, ln := d.inputText(b, s)
	// The line is vertically centered in the content box.
	top := cb.Y + (cb.H-ln.Height)/2
	x0 := cb.X - st.ScrollX
	if ph {
		x0 = cb.X
	}
	asc, desc := t.Ascent(), t.Descent()
	if st.Focused && !ph && st.Caret != st.Anchor {
		s0, s1 := min(st.Caret, st.Anchor), max(st.Caret, st.Anchor)
		xa, xb := t.X(ln, s0), t.X(ln, s1)
		c.FillRRect(paint.Rect(geom.Rect{X: x0 + xa, Y: top + ln.Baseline - asc, W: xb - xa, H: asc + desc}), selectionColor)
	}
	if s != "" {
		pc := color(col)
		if ph {
			pc = placeholderColor
		}
		_, size := t.RunFace(0)
		var run *paint.GlyphRun
		for _, g := range t.Glyphs(ln, nil) {
			if run == nil || run.Font != g.Face {
				if run != nil {
					c.Glyphs(run)
				}
				run = &paint.GlyphRun{Font: g.Face, Size: size, Color: pc}
			}
			run.Glyphs = append(run.Glyphs, paint.Glyph{ID: g.ID, X: x0 + g.X, Y: top + g.Y})
		}
		if run != nil {
			c.Glyphs(run)
		}
	}
	if st.Focused && st.Caret == st.Anchor {
		cx := geom.Unit(0)
		if !ph {
			cx = t.X(ln, st.Caret)
		} else {
			x0 = cb.X
		}
		c.FillRRect(paint.Rect(geom.Rect{X: x0 + cx, Y: top + ln.Baseline - asc, W: geom.Px(1), H: asc + desc}), color(col))
	}
}
