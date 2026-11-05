package text

import (
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Style is the part of a run's style that affects layout.
type Style struct {
	Weight     int
	Size       geom.Unit // font size
	LineHeight style.LineHeight
	// Lang is the content language (the inherited lang attribute, BCP 47;
	// "" for none). It selects language-specific glyph forms.
	Lang string
}

// Params are the properties of a paragraph: the style of the block that
// holds it (which also gives each line its strut, CSS 2.1 §10.8.1) and
// the block-level text properties (spec/SUBSET.md §7).
type Params struct {
	Style
	WhiteSpace style.WhiteSpace
	Align      style.TextAlign
	Ellipsis   bool // text-overflow: ellipsis applies (the box clips)
	RTL        bool // the paragraph's base direction is right to left
}

// Run is a piece of a paragraph with one style: the text of a text node
// in an inline element (or directly in the block), or a forced line break
// (<br>).
type Run struct {
	Text  string
	Style Style
	Break bool // a forced line break; Text is ignored
	// Outer are the styles of the inline boxes that enclose the run's own
	// box (outermost first). They take part in the height of every line
	// the run is on (CSS 2.1 §10.8).
	Outer []Style
}

// Text is a prepared paragraph: white space processed, shaped, and split
// into break opportunities. Advances are kept in "fine units" (font units
// x font size in 1/64 px, exact integers, so runs of different sizes add
// up exactly) and converted to geom.Unit only when compared or reported.
type Text struct {
	p    Params
	runs []Run
	rs   []runInfo // per run: face and metrics
	// runes after white-space processing. '\n' is a forced break, '\t' a
	// tab (only with white-space: pre / pre-wrap).
	runes []rune
	run   []int32 // run index of each rune
	adv   []int64 // advance of each rune in fine units (cluster advance on its first rune)
	// pos[i] is the pen position before rune i, in fine units from the
	// start of its hard line (tabs resolved there); len(pos) = len(runes)+1.
	pos []int64
	// Pieces (Chromium's inline items): maximal ranges of runes of one
	// run and one bidi level, split at tabs and forced breaks (not at
	// script changes: those split shaping, not the rounding of widths). pstart[i] and pend[i] bound
	// the piece of rune i. Widths are rounded per piece.
	pstart, pend []int
	// glyphs in logical order; Glyph.Cluster indexes runes.
	glyphs []Glyph
	// scr is the resolved script of each rune (UAX #24, common and
	// inherited characters take the script of their run); face is the
	// face its cluster was shaped with.
	scr  []language.Script
	face []*Face
	// lvl is the bidi embedding level of each rune; bidi is set when the
	// text has right-to-left (or mixed) levels.
	lvl  []uint8
	bidi bool
	// segs are the break opportunities: segment i spans
	// [segs[i].start, segs[i].end); its last `spaces` runes are spaces
	// that hang at a line end.
	segs []segment

	strut   runInfo // the block's own metrics
	tabStop int64   // fine units
	upem    int64

	minW, maxW geom.Unit
	last       *Block // cache of the last layout
	lastW      geom.Unit
}

type runInfo struct {
	outer      []runInfo
	face       *Face   // primary face
	chain      []*Face // fallback chain, starting with face
	lang       language.Language
	size       geom.Unit
	ascent     geom.Unit // rounded, as used for the baseline
	descent    geom.Unit
	lineHeight geom.Unit
}

// above and below return the run's extent above and below the baseline
// in a line box: its ascent plus half the leading (rounded down to a
// whole pixel, as Chromium does), and the rest of its line height.
func (r *runInfo) above() geom.Unit {
	l := r.lineHeight - r.ascent - r.descent
	return r.ascent + geom.Unit(floorDiv(int64(l), 2*geom.UnitsPerPx))*geom.UnitsPerPx
}
func (r *runInfo) below() geom.Unit { return r.lineHeight - r.above() }

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
	Height     geom.Unit // line box height
	Baseline   geom.Unit // offset of the baseline from the content box top
	// Ellipsis is set when the line was truncated: content is
	// [Start, End) and an ellipsis glyph follows at X+Width. FullEnd is
	// the end before truncation (the hidden content still has positions).
	Ellipsis bool
	FullEnd  int
	// endAdj corrects the width of [Start, End) when the line's last piece
	// was shaped again without the trailing spaces it was shaped with (see
	// reshapeEnd); it applies to positions at End.
	endAdj geom.Unit
	// vis is the visual order of the line (bidi text only, see reorder).
	vis []vseg
}

// New prepares the single-style paragraph s.
func New(s string, p Params) *Text {
	return NewRuns([]Run{{Text: s, Style: p.Style}}, p)
}

// NewRuns prepares a paragraph made of runs (raw text; white space is
// processed here, across run boundaries).
func NewRuns(runs []Run, p Params) *Text {
	t := &Text{p: p, runs: runs}
	t.strut = info(p.Style)
	t.upem = 1 << 16 / geom.UnitsPerPx // fine units are 1/65536 px
	for _, r := range runs {
		ri := info(r.Style)
		for _, o := range r.Outer {
			ri.outer = append(ri.outer, info(o))
		}
		t.rs = append(t.rs, ri)
	}
	t.processWhiteSpace()
	t.levels()
	t.shape()
	t.positions()
	t.pieces()
	t.segment()
	t.intrinsic()
	return t
}

// Empty reports whether the paragraph has no characters after white-space
// processing (such paragraphs are not rendered).
func (t *Text) Empty() bool { return len(t.runes) == 0 }

// String returns the processed text.
func (t *Text) String() string { return string(t.runes) }

// Params returns the paragraph's parameters.
func (t *Text) Params() Params { return t.p }

// RunOf returns the index of the run that rune i belongs to.
func (t *Text) RunOf(i int) int { return int(t.run[i]) }

// RunFace returns the face and size of run i.
func (t *Text) RunFace(i int) (*Face, geom.Unit) { return t.rs[i].face, t.rs[i].size }

func isCollapsible(ws style.WhiteSpace) bool {
	return ws == style.WhiteSpaceNormal || ws == style.WhiteSpaceNowrap
}

func wraps(ws style.WhiteSpace) bool {
	return ws == style.WhiteSpaceNormal || ws == style.WhiteSpacePreWrap
}

func info(s Style) runInfo {
	f := ForWeight(s.Weight)
	r := faceInfo(f, s)
	r.chain = Chain(s.Weight)
	r.lang = language.NewLanguage("en")
	if s.Lang != "" {
		r.lang = language.NewLanguage(s.Lang)
	}
	return r
}

// faceInfo returns the metrics of face f in style s.
func faceInfo(f *Face, s Style) runInfo {
	r := runInfo{face: f, size: s.Size}
	round := func(fu int32) geom.Unit {
		num := int64(fu) * int64(s.Size) // 1/64 px * upem
		den := int64(f.Upem) * geom.UnitsPerPx
		return geom.Px(int((num + den/2) / den))
	}
	r.ascent, r.descent = round(f.Ascent), round(f.Descent)
	switch lh := s.LineHeight; lh.Kind {
	case style.LineHeightNormal:
		r.lineHeight = r.ascent + r.descent + round(f.LineGap)
	case style.LineHeightNumber:
		r.lineHeight = geom.Unit(geom.MulDiv(int64(lh.V), int64(s.Size), 1000))
	default:
		r.lineHeight = geom.Unit(lh.V)
	}
	return r
}

// processWhiteSpace applies CSS Text 3 §4.1.1 phase I and the line-start
// / line-end removal of phase II that does not depend on wrapping, across
// the runs. With normal / nowrap: tabs and newlines become spaces, runs of
// spaces collapse to one (also across run boundaries; the space stays in
// the earlier run), and spaces at the start and end of the paragraph and
// around forced breaks are removed. With pre / pre-wrap everything is
// preserved; "\r\n" and "\r" become "\n", and a final newline does not
// start an empty line.
func (t *Text) processWhiteSpace() {
	collapse := isCollapsible(t.p.WhiteSpace)
	space := true // at the start, so leading spaces are dropped
	push := func(r rune, ri int) {
		t.runes = append(t.runes, r)
		t.run = append(t.run, int32(ri))
	}
	trim := func() {
		if n := len(t.runes); n > 0 && t.runes[n-1] == ' ' {
			t.runes, t.run = t.runes[:n-1], t.run[:n-1]
		}
	}
	for ri, run := range t.runs {
		if run.Break {
			if collapse {
				trim()
			}
			push('\n', ri)
			space = true
			continue
		}
		rs := []rune(run.Text)
		for i := 0; i < len(rs); i++ {
			r := rs[i]
			if collapse {
				switch r {
				case ' ', '\t', '\n', '\r', '\f':
					if !space {
						push(' ', ri)
						space = true
					}
				default:
					push(r, ri)
					space = false
				}
				continue
			}
			if r == '\r' {
				if i+1 < len(rs) && rs[i+1] == '\n' {
					continue
				}
				r = '\n'
			}
			if r == '\f' {
				r = ' '
			}
			push(r, ri)
		}
	}
	if collapse {
		trim()
	} else if n := len(t.runes); n > 0 && t.runes[n-1] == '\n' && !t.runs[t.run[n-1]].Break {
		t.runes, t.run = t.runes[:n-1], t.run[:n-1]
	}
}

// toUnits converts fine units to 1/64 px, rounding up.
func (t *Text) toUnits(f int64) geom.Unit {
	if f <= 0 {
		return -geom.Unit((-f) / t.upem)
	}
	return geom.Unit((f + t.upem - 1) / t.upem)
}

// toUnitsFloor converts fine units to 1/64 px rounding down (for glyph
// positions, which are relative and must not drift).
func (t *Text) toUnitsFloor(f int64) geom.Unit {
	return geom.Unit(floorDiv(f, t.upem))
}

// Ascent and Descent return the rounded ascent and descent of the
// paragraph's block style (the height of a text fragment is their sum).
func (t *Text) Ascent() geom.Unit  { return t.strut.ascent }
func (t *Text) Descent() geom.Unit { return t.strut.descent }

// RunMetrics returns the rounded ascent and descent of run i.
func (t *Text) RunMetrics(i int) (ascent, descent geom.Unit) { return t.rs[i].ascent, t.rs[i].descent }

// Runes returns the processed text of [s, e).
func (t *Text) Runes(s, e int) string { return string(t.runes[s:e]) }

// LineHeight returns the height of the block's strut (one line of its own
// font).
func (t *Text) LineHeight() geom.Unit { return t.strut.lineHeight }

// sameFont reports whether runs a and b shape alike: same fallback
// chain, size and language.
func (t *Text) sameFont(a, b int32) bool {
	return t.rs[a].face == t.rs[b].face && t.rs[a].size == t.rs[b].size && t.rs[a].lang == t.rs[b].lang
}

// itemize resolves the script of every rune (UAX #24, as Chromium's
// ScriptRunIterator): characters of the Common and Inherited scripts
// (spaces, digits, punctuation, combining marks, joiners) take the script
// of the run they are in (forced breaks do not end a run: they are
// Common characters too); at the start of the text they take the first
// strong script after them; a closing bracket takes the script of its
// opening bracket.
func (t *Text) itemize() {
	t.scr = make([]language.Script, len(t.runes))
	type open struct {
		r   rune
		scr language.Script
	}
	var stack []open
	cur := language.Common
	start := 0 // first rune still without a strong script
	for i, r := range t.runes {
		sc := language.LookupScript(r)
		if !sc.Strong() {
			if o, ok := bracket(r); ok {
				if o == 0 {
					stack = append(stack, open{r, cur})
				} else {
					j := len(stack) - 1
					for j >= 0 && stack[j].r != o {
						j--
					}
					if j >= 0 {
						sc = stack[j].scr
						stack = stack[:j]
					}
				}
			}
		}
		if sc.Strong() && sc != cur {
			if cur == language.Common {
				for k := start; k < i; k++ {
					t.scr[k] = sc
				}
				for k := range stack {
					stack[k].scr = sc
				}
			}
			cur = sc
		}
		t.scr[i] = cur
		if cur == language.Common {
			continue
		}
		start = i + 1
	}
	for k := start; k < len(t.runes); k++ {
		t.scr[k] = cur
	}
}

// shape shapes each item: a maximal range of one run style (chain, size,
// language) and one script between forced breaks and tabs. Like
// Chromium, shaping (kerning, ligatures) continues across inline element
// boundaries when the font is the same. Font fallback follows Chromium's
// HarfBuzzShaper: the item is shaped with the first face of the chain;
// each range of clusters containing a .notdef glyph is shaped again with
// the next face, and so on; the last face keeps its .notdef glyphs.
func (t *Text) shape() {
	t.itemize()
	t.adv = make([]int64, len(t.runes))
	t.face = make([]*Face, len(t.runes))
	start := 0
	for i := 0; i <= len(t.runes); i++ {
		if i < len(t.runes) && t.runes[i] != '\n' && t.runes[i] != '\t' && t.sameFont(t.run[i], t.run[start]) && t.scr[i] == t.scr[start] && t.lvl[i] == t.lvl[start] {
			continue
		}
		if i > start && t.runes[start] != '\n' && t.runes[start] != '\t' {
			ri := t.run[start]
			n := len(t.glyphs)
			t.glyphs = t.shapeFallback(start, i, t.rs[ri].chain, shapeProps{t.scr[start], t.rs[ri].lang, t.lvl[start]%2 == 1}, t.rs[ri].size, t.glyphs)
			for k := n; k < len(t.glyphs); k++ {
				g := &t.glyphs[k]
				t.adv[g.Cluster] += g.Advance
			}
			// A cluster spanning a run boundary (a ligature) keeps its
			// advance on its first rune; pieces split there regardless.
		}
		start = i
		if i < len(t.runes) && (t.runes[i] == '\n' || t.runes[i] == '\t') {
			start = i + 1
		}
	}
	for k := range t.glyphs { // the face of each cluster, on all its runes
		g := &t.glyphs[k]
		for j := g.Cluster; j < len(t.runes) && (j == g.Cluster || t.face[j] == nil && !t.clusterStart(j)); j++ {
			t.face[j] = g.Face
		}
	}
	f := t.strut.face
	if sp, ok := f.face.NominalGlyph(' '); ok {
		t.tabStop = 8 * f.ftAdvance(int64(f.face.HorizontalAdvance(sp)), t.strut.size)
	}
}

// shapeFallback shapes runes [s, e) with chain[0], re-shaping the ranges
// it has no glyphs for with the rest of the chain. Glyphs are appended in
// logical (cluster) order.
func (t *Text) shapeFallback(s, e int, chain []*Face, p shapeProps, size geom.Unit, out []Glyph) []Glyph {
	gs := chain[0].shape(t.runes, s, e, p, size, nil)
	if len(chain) == 1 {
		return append(out, gs...)
	}
	// Clusters (in logical order) and whether each lacks a glyph.
	missing := map[int]bool{}
	for _, g := range gs {
		if g.ID == 0 {
			missing[g.Cluster] = true
		}
	}
	if len(missing) == 0 {
		return append(out, gs...)
	}
	end := func(k int) int { // end of the cluster starting at gs[k]
		for j := k + 1; j < len(gs); j++ {
			if gs[j].Cluster != gs[k].Cluster {
				return gs[j].Cluster
			}
		}
		return e
	}
	for k := 0; k < len(gs); {
		c := gs[k].Cluster
		if !missing[c] {
			out = append(out, gs[k])
			k++
			continue
		}
		// A maximal range of missing clusters [c, ce).
		j := k
		for j < len(gs) && missing[gs[j].Cluster] {
			j++
		}
		ce := e
		if j < len(gs) {
			ce = gs[j].Cluster
		} else {
			ce = end(j - 1)
		}
		out = t.shapeFallback(c, ce, chain[1:], p, size, out)
		k = j
	}
	return out
}

// segment finds break opportunities: UAX #14 (go-text's segmenter),
// except between two printable ASCII characters, where Chromium uses its
// own table (asciiBreak) and a rule for '-' before a digit.
func (t *Text) segment() {
	n := len(t.runes)
	if n == 0 {
		return
	}
	brk := make([]bool, n+1) // a segment ends before rune p
	var seg segmenter.Segmenter
	seg.Init(t.runes)
	it := seg.LineIterator()
	for it.Next() {
		l := it.Line()
		brk[l.Offset+len(l.Text)] = true
	}
	ascii := func(r rune) bool { return r >= '!' && r <= '~' }
	for p := 1; p < n; p++ {
		a, b := t.runes[p-1], t.runes[p]
		if !ascii(a) || !ascii(b) {
			continue
		}
		if a == '-' && b >= '0' && b <= '9' {
			// "-1" may be a minus sign: break only in "ABCD-1234".
			brk[p] = p >= 2 && isASCIIAlnum(t.runes[p-2])
			continue
		}
		k := b - '!'
		brk[p] = asciiBreak[a-'!'][k/8]&(1<<(k%8)) != 0
	}
	for p := 1; p < n; p++ {
		if t.runes[p-1] == '\n' {
			brk[p] = true // forced
		}
	}
	start := 0
	for p := 1; p <= n; p++ {
		if !brk[p] {
			continue
		}
		s := segment{start: start, end: p}
		start = p
		// A forced break: the '\n' itself ends the segment.
		if t.runes[s.end-1] == '\n' {
			s.forced = true
			s.end--
		}
		for k := s.end - 1; k >= s.start && t.runes[k] == ' '; k-- {
			s.spaces++
		}
		// Without soft wrapping, only forced breaks end a segment.
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

func isASCIIAlnum(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

// width returns the advance of runes [s, e) placed at pen position x
// (fine units, for tab stops).
func (t *Text) width(s, e int, x int64) int64 {
	w := int64(0)
	for i := s; i < e; i++ {
		if t.runes[i] == '\t' {
			if t.tabStop > 0 {
				p := x + w
				w += t.tabStop - p%t.tabStop
			}
			continue
		}
		w += t.adv[i]
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
			x += t.adv[i]
		}
	}
	t.pos[len(t.runes)] = x
}

func (t *Text) pieces() {
	n := len(t.runes)
	t.pstart, t.pend = make([]int, n), make([]int, n)
	for i := 0; i < n; {
		j := i + 1
		if t.runes[i] != '\t' && t.runes[i] != '\n' {
			for j < n && t.run[j] == t.run[i] && t.lvl[j] == t.lvl[i] && t.runes[j] != '\t' && t.runes[j] != '\n' {
				j++
			}
		}
		for k := i; k < j; k++ {
			t.pstart[k], t.pend[k] = i, j
		}
		i = j
	}
}

// lineSpan returns the width of runes [s, e) of one line, starting a
// line, as Chromium computes it: the sum over the pieces (inline items)
// in the range of their exact advance rounded up to 1/64 px.
func (t *Text) lineSpan(s, e int) geom.Unit {
	var w geom.Unit
	for i := s; i < e; {
		j := min(t.pend[i], e)
		w += t.toUnits(t.pos[j] - t.pos[i])
		i = j
	}
	return w
}

// snapped returns the position of rune i in its hard line (which starts
// at rune line) with every piece before it rounded up, and the part in
// its own piece rounded up too.
func (t *Text) snapped(line, i int) geom.Unit {
	if i >= len(t.runes) {
		return t.lineSpan(line, i)
	}
	ps := max(t.pstart[i], line)
	return t.lineSpan(line, ps) + t.toUnits(t.pos[i]-t.pos[ps])
}

// span returns the width of runes [s, e) of the hard line starting at
// rune line, as Chromium measures it for min-content contributions: both
// ends are snapped up to 1/64 px in the coordinates of the hard line, and
// the width is their difference. A word in the middle of a line can
// therefore count 1/64 px less than lineSpan of the same word.
func (t *Text) span(line, s, e int) geom.Unit {
	if e <= s {
		return 0
	}
	return t.snapped(line, e) - t.snapped(line, s)
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
		t.minW = geom.Max(t.minW, t.span(line, s.start, s.end-t.hang(i)))
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

// Layout breaks the paragraph into lines for a content box of width w and
// returns them. The result is cached for the last width.
func (t *Text) Layout(w geom.Unit) *Block {
	if t.last != nil && t.lastW == w {
		return t.last
	}
	b := &Block{Width: w}
	wrap := wraps(t.p.WhiteSpace)
	limit := len(t.segs) // the line may take segments before limit
	for i := 0; i < len(t.segs); {
		// One line: take segments greedily.
		start := i
		ls := t.segs[start].start
		var content geom.Unit
		for i < limit {
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
		ln.FullEnd = ln.End
		if t.p.Align != style.TextAlignLeft && !last.forced && i < len(t.segs) && ln.End < last.end {
			kerned := ln
			t.reshapeEnd(&ln)
			// Without the kerning against the removed space the line can
			// overflow although it fitted: then, as Chromium's line breaker
			// does, its last segment moves to the next line, or with a
			// single segment the line keeps its kerned width.
			if wrap && ln.Width > w && kerned.Width <= w {
				if i-1 > start {
					i, limit = start, i-1
					continue
				}
				ln = kerned
			}
		}
		limit = len(t.segs)
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
		t.reorder(&ln)
		b.Lines = append(b.Lines, ln)
	}
	var y geom.Unit
	for k := range b.Lines {
		ln := &b.Lines[k]
		above, below := t.lineExtent(ln)
		ln.Top, ln.Height, ln.Baseline = y, above+below, y+above
		y += ln.Height
	}
	b.Height = y
	t.last, t.lastW = b, w
	return b
}

// reshapeEnd emulates Chromium removing the collapsible spaces at the end
// of a soft-wrapped line with text-align center or right
// (NGLineBreaker::RemoveTrailingCollapsibleSpace): the part of the line's
// last piece before the spaces is shaped again on its own, which drops
// kerning with the space (e.g. "." or "," followed by a space in Inter).
func (t *Text) reshapeEnd(ln *Line) {
	e := ln.End
	if e <= ln.Start {
		return
	}
	ps := max(t.pstart[e-1], ln.Start)
	ri := t.run[ps]
	var sum int64
	for s := ps; s < e; {
		k := s + 1
		for k < e && t.scr[k] == t.scr[s] && t.lvl[k] == t.lvl[s] {
			k++
		}
		for _, g := range t.shapeFallback(s, k, t.rs[ri].chain, shapeProps{t.scr[s], t.rs[ri].lang, t.lvl[s]%2 == 1}, t.rs[ri].size, nil) {
			sum += g.Advance
		}
		s = k
	}
	w := t.lineSpan(ln.Start, ps) + t.toUnits(sum)
	ln.endAdj = w - ln.Width
	ln.Width = w
}

// lineExtent returns how far a line box extends above and below its
// baseline: the maximum over the block's strut and the runs with content
// on the line (CSS 2.1 §10.8, all inline boxes baseline-aligned).
func (t *Text) lineExtent(ln *Line) (above, below geom.Unit) {
	above, below = t.strut.above(), t.strut.below()
	add := func(r *runInfo) {
		above, below = geom.Max(above, r.above()), geom.Max(below, r.below())
		for k := range r.outer {
			above, below = geom.Max(above, r.outer[k].above()), geom.Max(below, r.outer[k].below())
		}
	}
	seen := -1
	var seenFace *Face
	end := ln.FullEnd
	if end < len(t.runes) && t.runes[end] == '\n' && t.runs[t.run[end]].Break {
		end++ // a <br> ending the line is on it
	}
	for i := ln.Start; i < end; i++ {
		r := int(t.run[i])
		if r != seen {
			seen, seenFace = r, nil
			add(&t.rs[r])
		}
		// With line-height: normal, fallback fonts used on the line take
		// part with their own metrics (as in Chromium).
		if f := t.face[i]; f != nil && f != t.rs[r].face && f != seenFace && t.runs[r].Style.LineHeight.Kind == style.LineHeightNormal {
			seenFace = f
			fi := faceInfo(f, t.runs[r].Style)
			add(&fi)
		}
	}
	return
}

// Height returns the content height at width w.
func (t *Text) Height(w geom.Unit) geom.Unit {
	if !wraps(t.p.WhiteSpace) {
		w = t.maxW // no soft breaks: the width does not matter
	}
	return t.Layout(w).Height
}

// truncate shortens ln so that its content plus an ellipsis fits in w
// (CSS Overflow 3 §3.2): whole clusters are removed from the end, and
// spaces before the ellipsis are kept.
func (t *Text) truncate(ln *Line, w geom.Unit) {
	ell := t.ellipsisAdvance()
	end := ln.End
	for end > ln.Start {
		if t.lineSpan(ln.Start, end)+t.toUnits(ell) <= w {
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

// ellipsisAdvance is the advance of "…" in the block's style, fine units.
func (t *Text) ellipsisAdvance() int64 {
	return t.strut.face.ftAdvance(int64(t.strut.face.ellipsisAdvance()), t.strut.size)
}

// InkWidth returns the width of ln without trailing spaces.
func (t *Text) InkWidth(ln *Line) geom.Unit {
	e := ln.End
	for e > ln.Start && t.runes[e-1] == ' ' && t.p.WhiteSpace != style.WhiteSpacePre {
		e--
	}
	if e == ln.End {
		return t.lineSpan(ln.Start, e) + ln.endAdj
	}
	return t.lineSpan(ln.Start, e)
}

// EllipsisWidth returns the advance of the ellipsis glyph.
func (t *Text) EllipsisWidth() geom.Unit { return t.toUnits(t.ellipsisAdvance()) }

// X returns the x offset of the pen before rune i on line ln (relative to
// the content box), for fragments and hit testing.
func (t *Text) X(ln *Line, i int) geom.Unit {
	if ln.vis != nil {
		return t.visX(ln, i)
	}
	if i >= len(t.runes) || i >= ln.FullEnd {
		if i == ln.End {
			return ln.X + t.lineSpan(ln.Start, i) + ln.endAdj
		}
		return ln.X + t.lineSpan(ln.Start, i)
	}
	ps := max(t.pstart[i], ln.Start)
	return ln.X + t.lineSpan(ln.Start, ps) + t.toUnitsFloor(t.pos[i]-t.pos[ps])
}

// PlacedGlyph is a glyph positioned relative to the content box origin
// (x of the pen, y of the baseline), in Units. Run is the index of its
// run; Run < 0 is the ellipsis (block style).
type PlacedGlyph struct {
	ID   uint16
	X, Y geom.Unit
	Run  int
	Face *Face
}

// Glyphs appends the glyphs of line ln (including its ellipsis) with
// positions relative to the content box.
func (t *Text) Glyphs(ln *Line, out []PlacedGlyph) []PlacedGlyph {
	if ln.vis != nil {
		out = t.visualGlyphs(ln, out)
		if ln.Ellipsis {
			if id, ok := t.strut.face.face.NominalGlyph('…'); ok {
				x := ln.X + ln.Width
				if t.p.RTL {
					x = ln.X - t.EllipsisWidth()
				}
				out = append(out, PlacedGlyph{ID: uint16(id), X: x, Y: ln.Baseline, Run: -1, Face: t.strut.face})
			}
		}
		return out
	}
	gi := 0
	for gi < len(t.glyphs) && t.glyphs[gi].Cluster < ln.Start {
		gi++
	}
	for i := ln.Start; i < ln.End; i++ {
		var off int64
		ps := max(t.pstart[i], ln.Start)
		px := ln.X + t.lineSpan(ln.Start, ps)
		for ; gi < len(t.glyphs) && t.glyphs[gi].Cluster == i; gi++ {
			g := t.glyphs[gi]
			r := t.run[i]
			out = append(out, PlacedGlyph{
				ID:   g.ID,
				X:    px + t.toUnitsFloor(t.pos[i]-t.pos[ps]+off+g.XOff),
				Y:    ln.Baseline - t.toUnitsFloor(g.YOff),
				Run:  int(r),
				Face: g.Face,
			})
			off += g.Advance
		}
	}
	if ln.Ellipsis {
		if id, ok := t.strut.face.face.NominalGlyph('…'); ok {
			out = append(out, PlacedGlyph{ID: uint16(id), X: ln.X + t.lineSpan(ln.Start, ln.End), Y: ln.Baseline, Run: -1, Face: t.strut.face})
		}
	}
	return out
}

// StrutFace returns the block's face and size (used for the ellipsis).
func (t *Text) StrutFace() (*Face, geom.Unit) { return t.strut.face, t.strut.size }

func (f *Face) ellipsisAdvance() int32 {
	if id, ok := f.face.NominalGlyph('…'); ok {
		return int32(f.face.HorizontalAdvance(id))
	}
	return 0
}

// Metrics returns the rounded ascent and descent of a style's font (the
// content area of an inline box, CSS 2.1 §10.6.1).
func Metrics(s Style) (ascent, descent geom.Unit) {
	r := info(s)
	return r.ascent, r.descent
}

// RuneRange returns the runes [s, e) that belong to runs [first, last).
func (t *Text) RuneRange(first, last int) (s, e int) {
	s, e = -1, -1
	for i, r := range t.run {
		if int(r) >= first && int(r) < last {
			if s < 0 {
				s = i
			}
			e = i + 1
		}
	}
	if s < 0 {
		// No characters (e.g. an empty element): the position of the
		// next rune after the runs before it.
		p := 0
		for p < len(t.run) && int(t.run[p]) < first {
			p++
		}
		return p, p
	}
	return s, e
}

// LineInk returns the top and bottom of the content areas of the
// characters on line ln (relative to the content box).
func (t *Text) LineInk(ln *Line) (top, bottom geom.Unit) {
	top, bottom = ln.Baseline-t.strut.ascent, ln.Baseline+t.strut.descent
	if ln.End > ln.Start {
		top, bottom = 1<<40, -1<<40
	}
	for i := ln.Start; i < ln.End; i++ {
		r := &t.rs[t.run[i]]
		top, bottom = min(top, ln.Baseline-r.ascent), max(bottom, ln.Baseline+r.descent)
	}
	return
}

// LineHeightOf returns the line height of one line of a style's font.
func LineHeightOf(s Style) geom.Unit { return info(s).lineHeight }
