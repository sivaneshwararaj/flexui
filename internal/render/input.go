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
	// MarkStart, MarkEnd: the IME marked text (rune indices), underlined;
	// equal when there is no composition.
	MarkStart, MarkEnd int
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

// inputText returns the laid-out single line of s in the input's style
// and direction (bidi reordered; positions from the line's left edge).
func (d *Document) inputText(b *Box, s string) (*text.Text, *text.Line) {
	p := text.Params{Style: textStyle(&b.Style), WhiteSpace: style.WhiteSpacePre,
		RTL: b.Style.Direction == style.DirectionRTL}
	t := d.cache.text([]text.Run{{Text: s, Style: p.Style}}, p)
	blk := t.Layout(1 << 40)
	if len(blk.Lines) == 0 {
		return t, &text.Line{Baseline: t.Ascent(), Height: t.LineHeight()}
	}
	return t, &blk.Lines[0]
}

// inputOffset is where the line of text starts in the content box before
// scrolling: 0 in a left-to-right input; in a right-to-left one the text
// is right-aligned (negative when it is wider than the box).
func (d *Document) inputOffset(b *Box, ln *text.Line) geom.Unit {
	if b.Style.Direction != style.DirectionRTL {
		return 0
	}
	return b.contentBox().W - ln.Width
}

// InputCaretX returns the x offset of the caret before rune i in the
// input's text (relative to its content box, unscrolled; bidi aware), and
// the width of the content box.
func (d *Document) InputCaretX(n *dom.Node, s string, i int) (x, width geom.Unit) {
	b := d.boxes[n]
	if b == nil || b.Lay == nil {
		return 0, 0
	}
	t, ln := d.inputText(b, s)
	off := d.inputOffset(b, ln)
	if len([]rune(s)) == 0 {
		return off, b.contentBox().W
	}
	return off + t.X(ln, i), b.contentBox().W
}

// InputCaretRect returns the caret rectangle of input n with text s and
// the caret before rune i, scrolled by scrollX, in document coordinates.
func (d *Document) InputCaretRect(n *dom.Node, s string, i int, scrollX geom.Unit) geom.Rect {
	b := d.boxes[n]
	if b == nil || b.Lay == nil {
		return geom.Rect{}
	}
	cb := b.contentBox()
	t, ln := d.inputText(b, s)
	x, _ := d.InputCaretX(n, s, i)
	top := cb.Y + (cb.H-ln.Height)/2
	asc, desc := t.Ascent(), t.Descent()
	return geom.Rect{X: cb.X - scrollX + x, Y: top + ln.Baseline - asc, W: geom.Px(1), H: asc + desc}
}

// InputScrollRange returns the range of horizontal scroll offsets that
// keep the text of s in view in an input whose usable width is room.
func (d *Document) InputScrollRange(n *dom.Node, s string, room geom.Unit) (lo, hi geom.Unit) {
	b := d.boxes[n]
	if b == nil || b.Lay == nil || s == "" {
		return 0, 0
	}
	_, ln := d.inputText(b, s)
	off := d.inputOffset(b, ln)
	return min(0, off), max(0, off+ln.Width-room)
}

// InputRTL reports whether input n is right to left (its arrow keys move
// the other way, as in Chromium).
func (d *Document) InputRTL(n *dom.Node) bool {
	b := d.boxes[n]
	return b != nil && b.Style.Direction == style.DirectionRTL
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
	off := d.inputOffset(b, ln)
	x0 := cb.X - st.ScrollX + off
	if ph {
		x0 = cb.X + max(0, off)
	}
	asc, desc := t.Ascent(), t.Descent()
	if st.Focused && !ph && st.Caret != st.Anchor {
		// In bidi text a selection can be several rectangles.
		s0, s1 := min(st.Caret, st.Anchor), max(st.Caret, st.Anchor)
		for _, sp := range t.Spans(ln, s0, s1) {
			c.FillRRect(paint.Rect(geom.Rect{X: x0 + sp[0], Y: top + ln.Baseline - asc, W: sp[1] - sp[0], H: asc + desc}), selectionColor)
		}
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
	if !ph && st.MarkEnd > st.MarkStart {
		// The marked text of an IME composition: a 1px underline in the
		// text color, 2px below the baseline.
		for _, sp := range t.Spans(ln, st.MarkStart, st.MarkEnd) {
			c.FillRRect(paint.Rect(geom.Rect{X: x0 + sp[0], Y: top + ln.Baseline + geom.Px(2), W: sp[1] - sp[0], H: geom.Px(1)}), color(col))
		}
	}
	if st.Focused && st.Caret == st.Anchor {
		cx := geom.Unit(0)
		if !ph {
			cx = t.X(ln, st.Caret)
		} else {
			x0 = cb.X // an empty input: the caret at the start side
			if b.Style.Direction == style.DirectionRTL {
				x0 = cb.X + cb.W - geom.Px(1)
			}
		}
		c.FillRRect(paint.Rect(geom.Rect{X: x0 + cx, Y: top + ln.Baseline - asc, W: geom.Px(1), H: asc + desc}), color(col))
	}
}
