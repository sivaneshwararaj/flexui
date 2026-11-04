package text

import (
	"github.com/go-text/typesetting/segmenter"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Params are the resolved text properties of one anonymous text run
// (spec/SUBSET.md §7).
type Params struct {
	Weight     int
	Size       geom.Unit // font size
	LineHeight style.LineHeight
	WhiteSpace style.WhiteSpace
	Align      style.TextAlign
	Ellipsis   bool // text-overflow: ellipsis applies (the run clips)
}

// Text is a prepared text run: white space processed, shaped, and split
// into break opportunities. Widths are kept in font units (exact
// integers) and converted to geom.Unit only when compared or reported.
type Text struct {
	p    Params
	face *Face
	// runes after white-space processing. '\n' is a forced break, '\t' a
	// tab (only with white-space: pre / pre-wrap).
	runes []rune
	adv   []int32 // advance of each rune in font units (cluster advance on its first rune)
	// pos[i] is the pen position before rune i, in font units from the
	// start of its hard line (tabs resolved there); len(pos) = len(runes)+1.
	pos []int64
	// glyphs in logical order; Glyph.Cluster indexes runes.
	glyphs []Glyph
	// segs are the break opportunities: segment i spans
	// [segs[i].start, segs[i].end); its last `spaces` runes are spaces
	// that hang at a line end.
	segs []segment

	lineHeight geom.Unit
	ascent     geom.Unit // rounded, as used for the baseline
	descent    geom.Unit
	tabStop    int32 // font units

	minW, maxW geom.Unit
	last       *Block // cache of the last layout
	lastW      geom.Unit
}

type segment struct {
	start, end int
	spaces     int  // trailing hanging spaces
	forced     bool // a forced break follows this segment
}

// Block is the result of laying out a Text at a given width.
type Block struct {
	Lines  []Line
	Height geom.Unit
	Width  geom.Unit // the width it was laid out at
}

// Line is one line box.
type Line struct {
	Start, End int       // rune range, trailing hanging spaces excluded
	Width      geom.Unit // advance width of [Start, End)
	X          geom.Unit // offset from the content box left (text-align)
	Top        geom.Unit // offset of the line box from the content box top
	Baseline   geom.Unit // offset of the baseline from the content box top
	// Ellipsis is set when the line was truncated: content is
	// [Start, End) and an ellipsis glyph follows at X+Width.
	Ellipsis bool
}

// New prepares s with the parameters p. s is the raw text of the run
// (adjacent text nodes concatenated); white space is processed here.
func New(s string, p Params) *Text {
	t := &Text{p: p, face: ForWeight(p.Weight)}
	t.runes = processWhiteSpace(s, p.WhiteSpace)
	t.metrics()
	t.shape()
	t.positions()
	t.segment()
	t.intrinsic()
	return t
}

// Empty reports whether the run has no characters after white-space
// processing (such runs are not rendered).
func (t *Text) Empty() bool { return len(t.runes) == 0 }

// String returns the processed text.
func (t *Text) String() string { return string(t.runes) }

// Face returns the font face used for the run.
func (t *Text) Face() *Face { return t.face }

// Params returns the run's parameters.
func (t *Text) Params() Params { return t.p }

func isCollapsible(ws style.WhiteSpace) bool {
	return ws == style.WhiteSpaceNormal || ws == style.WhiteSpaceNowrap
}

func wraps(ws style.WhiteSpace) bool {
	return ws == style.WhiteSpaceNormal || ws == style.WhiteSpacePreWrap
}

// processWhiteSpace applies CSS Text 3 §4.1.1 phase I and the line-start
// / line-end removal of phase II that does not depend on wrapping. With
// normal / nowrap: tabs and newlines become spaces, runs of spaces collapse
// to one, and spaces at the start and end of the run are removed. With
// pre / pre-wrap everything is preserved; "\r\n" and "\r" become "\n".
func processWhiteSpace(s string, ws style.WhiteSpace) []rune {
	out := make([]rune, 0, len(s))
	if isCollapsible(ws) {
		space := true // at the start, so leading spaces are dropped
		for _, r := range s {
			switch r {
			case ' ', '\t', '\n', '\r', '\f':
				if !space {
					out = append(out, ' ')
					space = true
				}
			default:
				out = append(out, r)
				space = false
			}
		}
		if len(out) > 0 && out[len(out)-1] == ' ' {
			out = out[:len(out)-1]
		}
		return out
	}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if r == '\r' {
			if i+1 < len(rs) && rs[i+1] == '\n' {
				continue
			}
			r = '\n'
		}
		if r == '\f' {
			r = ' '
		}
		out = append(out, r)
	}
	// A final newline ends the last line; it does not start a new one.
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out
}

func (t *Text) toUnits(fu int64) geom.Unit {
	// ceil(fu * size / upem) in 1/64 px.
	n := int64(t.p.Size)
	if fu <= 0 {
		return -geom.Unit(geom.MulDiv(-fu, n, int64(t.face.Upem)))
	}
	return geom.Unit(geom.MulDiv(fu, n, int64(t.face.Upem)) + boolInt(geom.MulDiv(fu, n, int64(t.face.Upem))*int64(t.face.Upem) != fu*n))
}

// ToUnitsFloor converts font units to 1/64 px rounding down (for glyph
// positions, which are relative and must not drift).
func (t *Text) ToUnitsFloor(fu int64) geom.Unit {
	if fu < 0 {
		return -t.toUnits(-fu)
	}
	return geom.Unit(geom.MulDiv(fu, int64(t.p.Size), int64(t.face.Upem)))
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// roundFU returns fu*size/upem rounded to the nearest whole pixel (half
// up), in Units.
func (t *Text) roundFU(fu int32) geom.Unit {
	num := int64(fu) * int64(t.p.Size) // 1/64 px * upem
	den := int64(t.face.Upem) * geom.UnitsPerPx
	return geom.Px(int((num + den/2) / den))
}

func (t *Text) metrics() {
	f := t.face
	t.ascent = t.roundFU(f.Ascent)
	t.descent = t.roundFU(f.Descent)
	gap := t.roundFU(f.LineGap)
	switch lh := t.p.LineHeight; lh.Kind {
	case style.LineHeightNormal:
		t.lineHeight = t.ascent + t.descent + gap
	case style.LineHeightNumber:
		t.lineHeight = geom.Unit(geom.MulDiv(int64(lh.V), int64(t.p.Size), 1000))
	default:
		t.lineHeight = geom.Unit(lh.V)
	}
	if sp, ok := f.face.NominalGlyph(' '); ok {
		t.tabStop = 8 * int32(f.face.HorizontalAdvance(sp))
	}
}

// Ascent and Descent return the rounded font ascent and descent used to
// place the baseline (the height of a text fragment is their sum).
func (t *Text) Ascent() geom.Unit  { return t.ascent }
func (t *Text) Descent() geom.Unit { return t.descent }

// Runes returns the processed text of [s, e).
func (t *Text) Runes(s, e int) string { return string(t.runes[s:e]) }

// LineHeight returns the height of one line box.
func (t *Text) LineHeight() geom.Unit { return t.lineHeight }

// shape shapes each piece of text between forced breaks and tabs.
func (t *Text) shape() {
	t.adv = make([]int32, len(t.runes))
	start := 0
	for i := 0; i <= len(t.runes); i++ {
		if i < len(t.runes) && t.runes[i] != '\n' && t.runes[i] != '\t' {
			continue
		}
		if i > start {
			n := len(t.glyphs)
			t.glyphs = t.face.shape(t.runes[start:i], t.glyphs)
			for k := n; k < len(t.glyphs); k++ {
				g := &t.glyphs[k]
				g.Cluster += start
				t.adv[g.Cluster] += g.Advance
			}
		}
		start = i + 1
	}
}

// segment finds break opportunities (UAX #14).
func (t *Text) segment() {
	if len(t.runes) == 0 {
		return
	}
	var seg segmenter.Segmenter
	seg.Init(t.runes)
	it := seg.LineIterator()
	for it.Next() {
		l := it.Line()
		s := segment{start: l.Offset, end: l.Offset + len(l.Text)}
		// A forced break: the '\n' itself ends the segment.
		if s.end > s.start && t.runes[s.end-1] == '\n' {
			s.forced = true
			s.end--
		}
		for k := s.end - 1; k >= s.start && t.runes[k] == ' '; k-- {
			s.spaces++
		}
		// Without soft wrapping, only forced breaks end a segment.
		// Tailoring to match Chromium (ICU): no break after '/' or '!'
		// directly followed by the next word.
		if n := len(t.segs); n > 0 && !t.segs[n-1].forced && t.segs[n-1].spaces == 0 &&
			(t.runes[t.segs[n-1].end-1] == '/' || t.runes[t.segs[n-1].end-1] == '!') {
			t.segs[n-1].end, t.segs[n-1].spaces, t.segs[n-1].forced = s.end, s.spaces, s.forced
			continue
		}
		if n := len(t.segs); n > 0 && !t.segs[n-1].forced && !wraps(t.p.WhiteSpace) {
			t.segs[n-1].end, t.segs[n-1].spaces, t.segs[n-1].forced = s.end, s.spaces, s.forced
			continue
		}
		t.segs = append(t.segs, s)
	}
	if t.p.WhiteSpace == style.WhiteSpacePre {
		// Preserved spaces do not hang with white-space: pre.
		for i := range t.segs {
			t.segs[i].spaces = 0
		}
	}
}

// width returns the advance of runes [s, e) placed at pen position x
// (font units, for tab stops).
func (t *Text) width(s, e int, x int64) int64 {
	w := int64(0)
	for i := s; i < e; i++ {
		if t.runes[i] == '\t' {
			if t.tabStop > 0 {
				p := x + w
				w += int64(t.tabStop) - p%int64(t.tabStop)
			}
			continue
		}
		w += int64(t.adv[i])
	}
	return w
}

func (t *Text) positions() {
	t.pos = make([]int64, len(t.runes)+1)
	var x int64
	for i, r := range t.runes {
		t.pos[i] = x
		switch r {
		case '\n':
			x = 0
		case '\t':
			x += t.width(i, i+1, x)
		default:
			x += int64(t.adv[i])
		}
	}
	t.pos[len(t.runes)] = x
}

// lineSpan returns the width of runes [s, e) of one line, starting a
// line: the exact advance rounded up to 1/64 px.
func (t *Text) lineSpan(s, e int) geom.Unit {
	if e <= s {
		return 0
	}
	return t.toUnits(t.pos[e] - t.pos[s])
}

// span returns the width of runes [s, e) of one hard line as Chromium
// measures it for min-content contributions: both ends are snapped up to
// 1/64 px in the coordinates of the hard line, and the width is their
// difference. A word in the middle of a line can therefore count 1/64 px
// less than lineSpan of the same word.
func (t *Text) span(s, e int) geom.Unit {
	if e <= s {
		return 0
	}
	return t.toUnits(t.pos[e]) - t.toUnits(t.pos[s])
}

// conditional reports whether the trailing spaces of segment i only
// conditionally hang (CSS Text 3 §4.1.3): with pre-wrap, spaces before a
// forced break or the end of the run hang only if they do not fit, so
// they count for intrinsic sizes and alignment. Spaces at a soft wrap
// opportunity always hang.
func (t *Text) conditional(i int) bool {
	return t.p.WhiteSpace == style.WhiteSpacePreWrap && (t.segs[i].forced || i == len(t.segs)-1)
}

// hang returns the number of trailing spaces of segment i that do not
// count for intrinsic sizes.
func (t *Text) hang(i int) int {
	if t.conditional(i) {
		return 0
	}
	return t.segs[i].spaces
}

func (t *Text) intrinsic() {
	line := 0 // first rune of the current hard line
	for i, s := range t.segs {
		if i == 0 || t.segs[i-1].forced {
			line = s.start
		}
		t.minW = geom.Max(t.minW, t.span(s.start, s.end-t.hang(i)))
		t.maxW = geom.Max(t.maxW, t.lineSpan(line, s.end-t.hang(i)))
	}
	if !wraps(t.p.WhiteSpace) {
		t.minW = t.maxW
	}
}

// MinContent returns the min-content width: the widest unbreakable piece.
func (t *Text) MinContent() geom.Unit { return t.minW }

// MaxContent returns the max-content width: the widest line when only
// forced breaks are taken.
func (t *Text) MaxContent() geom.Unit { return t.maxW }

// Layout breaks the run into lines for a content box of width w and
// returns them. The result is cached for the last width.
func (t *Text) Layout(w geom.Unit) *Block {
	if t.last != nil && t.lastW == w {
		return t.last
	}
	b := &Block{Width: w}
	wrap := wraps(t.p.WhiteSpace)
	for i := 0; i < len(t.segs); {
		// One line: take segments greedily.
		start := i
		ls := t.segs[start].start
		var content geom.Unit
		for i < len(t.segs) {
			s := t.segs[i]
			c := t.lineSpan(ls, s.end-s.spaces)
			if i > start && wrap && c > w {
				break
			}
			content = c
			i++
			if s.forced {
				break
			}
		}
		last := t.segs[i-1]
		ln := Line{Start: ls, End: last.end - last.spaces, Width: content}
		if t.conditional(i-1) && last.spaces > 0 {
			if full := t.lineSpan(ls, last.end); full <= w || !wrap {
				ln.End, ln.Width = last.end, full
			}
		}
		if t.p.Ellipsis && ln.Width > w {
			t.truncate(&ln, w)
		}
		switch t.p.Align {
		case style.TextAlignRight:
			ln.X = w - ln.Width
		case style.TextAlignCenter:
			ln.X = (w - ln.Width) / 2
		}
		if ln.X < 0 {
			ln.X = 0
		}
		b.Lines = append(b.Lines, ln)
	}
	half := t.halfLeading()
	for k := range b.Lines {
		b.Lines[k].Top = geom.Unit(k) * t.lineHeight
		b.Lines[k].Baseline = b.Lines[k].Top + t.ascent + half
	}
	b.Height = geom.Unit(len(b.Lines)) * t.lineHeight
	t.last, t.lastW = b, w
	return b
}

// halfLeading is half the difference between the line height and the
// font's ascent + descent, floored to a whole pixel (as Chromium does), so
// the baseline of every line is on a whole pixel when the line box is.
func (t *Text) halfLeading() geom.Unit {
	l := t.lineHeight - t.ascent - t.descent
	return geom.Unit(floorDiv(int64(l), 2*geom.UnitsPerPx)) * geom.UnitsPerPx
}

// Height returns the content height at width w.
func (t *Text) Height(w geom.Unit) geom.Unit {
	if !wraps(t.p.WhiteSpace) {
		return geom.Unit(len(t.hardLines())) * t.lineHeight
	}
	return t.Layout(w).Height
}

func (t *Text) hardLines() []int {
	var out []int
	for i, s := range t.segs {
		if i == 0 || t.segs[i-1].forced {
			out = append(out, s.start)
		}
	}
	return out
}

// truncate shortens ln so that its content plus an ellipsis fits in w
// (CSS Overflow 3 §3.2): whole clusters are removed from the end, and
// spaces before the ellipsis are kept.
func (t *Text) truncate(ln *Line, w geom.Unit) {
	ell := int64(t.face.ellipsisAdvance())
	end := ln.End
	for end > ln.Start {
		if t.toUnits(t.pos[end]-t.pos[ln.Start]+ell) <= w {
			break
		}
		end--
		for end > ln.Start && t.adv[end] == 0 && t.runes[end] != '\t' && !t.clusterStart(end) {
			end--
		}
	}
	ln.End = end
	ln.Width = t.lineSpan(ln.Start, end)
	ln.Ellipsis = true
}

func (t *Text) clusterStart(i int) bool {
	for _, g := range t.glyphs {
		if g.Cluster == i {
			return true
		}
	}
	return false
}

// InkWidth returns the width of ln without trailing spaces.
func (t *Text) InkWidth(ln *Line) geom.Unit {
	e := ln.End
	for e > ln.Start && t.runes[e-1] == ' ' && t.p.WhiteSpace != style.WhiteSpacePre {
		e--
	}
	return t.lineSpan(ln.Start, e)
}

// EllipsisWidth returns the advance of the ellipsis glyph.
func (t *Text) EllipsisWidth() geom.Unit { return t.toUnits(int64(t.face.ellipsisAdvance())) }

// PlacedGlyph is a glyph positioned relative to the content box origin
// (x of the pen, y of the baseline), in Units.
type PlacedGlyph struct {
	ID   uint16
	X, Y geom.Unit
}

// Glyphs appends the glyphs of line ln (including its ellipsis) with
// positions relative to the content box.
func (t *Text) Glyphs(ln *Line, out []PlacedGlyph) []PlacedGlyph {
	var pen int64
	gi := 0
	for gi < len(t.glyphs) && t.glyphs[gi].Cluster < ln.Start {
		gi++
	}
	for i := ln.Start; i < ln.End; i++ {
		if t.runes[i] == '\t' {
			pen += t.width(i, i+1, pen)
			continue
		}
		for ; gi < len(t.glyphs) && t.glyphs[gi].Cluster == i; gi++ {
			g := t.glyphs[gi]
			out = append(out, PlacedGlyph{
				ID: g.ID,
				X:  ln.X + t.ToUnitsFloor(pen+int64(g.XOff)),
				Y:  ln.Baseline - t.ToUnitsFloor(int64(g.YOff)),
			})
			pen += int64(g.Advance)
		}
	}
	if ln.Ellipsis {
		if id, ok := t.face.face.NominalGlyph('…'); ok {
			out = append(out, PlacedGlyph{ID: uint16(id), X: ln.X + t.ToUnitsFloor(pen), Y: ln.Baseline})
		}
	}
	return out
}

func (f *Face) ellipsisAdvance() int32 {
	if id, ok := f.face.NominalGlyph('…'); ok {
		return int32(f.face.HorizontalAdvance(id))
	}
	return 0
}
