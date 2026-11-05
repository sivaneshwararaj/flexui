package text

import (
	"reflect"
	"unsafe"

	"github.com/go-text/typesetting/bidi"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Bidirectional text (UAX #9, spec/SUBSET.md §7.6).
//
// Levels are resolved per hard line (a forced break is a paragraph
// separator, as in Chromium, which runs ICU's bidi over the block's text
// with '\n' as separator) with go-text's implementation, which gets a
// single paragraph right but drops the text after a separator. Items and
// pieces are split at level changes; odd levels are shaped right to left.
// Lines are reordered (rules L1, L2) into visual segments only when the
// text has a right-to-left level; all-LTR text keeps the plain paths.

func (t *Text) levels() {
	t.lvl = make([]uint8, len(t.runes))
	base := bidi.LeftToRight
	var para uint8
	if t.p.RTL {
		base, para = bidi.RightToLeft, 1
	}
	var p bidi.Paragraph
	for a := 0; a < len(t.runes); {
		b := a
		for b < len(t.runes) && t.runes[b] != '\n' {
			b++
		}
		if b > a {
			in, at := t.withIsolates(a, b)
			rs := p.Segment(in, base)
			lv := runLevels(&rs)
			for k := a; k < b; k++ {
				t.lvl[k] = uint8(lv[at[k-a]])
			}
			// The isolate controls stay in the reordering as zero-width
			// separators with their levels (as in Chromium's text).
			prev := 0
			for k := a; k <= b; k++ {
				end := len(in)
				if k < b {
					end = at[k-a]
				}
				for j := prev; j < end; j++ {
					if t.ctl == nil {
						t.ctl = map[int][]uint8{}
					}
					t.ctl[k] = append(t.ctl[k], uint8(lv[j]))
				}
				prev = end + 1
			}
		}
		if b < len(t.runes) {
			t.lvl[b] = para
		}
		a = b + 1
	}
	for _, l := range t.lvl {
		if l != t.lvl[0] || l%2 == 1 {
			t.bidi = true
			break
		}
	}
}

// isoChain returns the isolates (outermost first) that rune i is in.
func (t *Text) isoChain(i int) []Style {
	r := &t.runs[t.run[i]]
	var c []Style
	for _, o := range r.Outer {
		if o.IsoID > 0 {
			c = append(c, o)
		}
	}
	if r.Style.IsoID > 0 {
		c = append(c, r.Style)
	}
	return c
}

// withIsolates returns runes [a, b) with isolate controls (LRI, RLI or
// FSI ... PDI) around the text of inline elements with a dir attribute,
// and the index in it of every rune.
func (t *Text) withIsolates(a, b int) ([]rune, []int) {
	at := make([]int, b-a)
	var in []rune
	var open []Style
	for i := a; i < b; i++ {
		c := t.isoChain(i)
		k := 0
		for k < len(open) && k < len(c) && open[k].IsoID == c[k].IsoID {
			k++
		}
		for len(open) > k {
			in = append(in, '\u2069') // PDI
			open = open[:len(open)-1]
		}
		for _, s := range c[k:] {
			in = append(in, [...]rune{0, '\u2066', '\u2067', '\u2068'}[s.IsoDir]) // LRI, RLI, FSI
			open = append(open, s)
		}
		at[i-a] = len(in)
		in = append(in, t.runes[i])
	}
	return in, at
}

// runLevels returns the resolved level of every character. go-text's
// Runs only split where the direction changes and report the level of
// each run's first character (levels 0 and 2 merge, e.g. Arabic-Indic
// digits in left-to-right text), but keep the per-character levels in an
// unexported field, read here (TestRunLevels guards it; this and
// coverage.go are the module's only uses of unsafe).
func runLevels(rs *bidi.Runs) []bidi.Level {
	f := reflect.ValueOf(rs).Elem().FieldByName("levels")
	return *(*[]bidi.Level)(unsafe.Pointer(f.UnsafeAddr()))
}

// vseg is a visual segment of a line: runes [s, e) of one piece and one
// level, at x (from the content box left) with width w.
type vseg struct {
	s, e int
	rtl  bool
	x, w geom.Unit
}

// reorder computes ln.vis: the line's runes [Start, End) cut at piece and
// level boundaries, in visual order (L1: trailing white space takes the
// paragraph level; L2: reverse sequences from the highest level down to
// the lowest odd level).
func (t *Text) reorder(ln *Line) {
	if !t.bidi || ln.End <= ln.Start {
		return
	}
	var para uint8
	if t.p.RTL {
		para = 1
	}
	lv := make([]uint8, ln.End-ln.Start)
	copy(lv, t.lvl[ln.Start:ln.End])
	for k := len(lv) - 1; k >= 0 && (t.runes[ln.Start+k] == ' ' || t.runes[ln.Start+k] == '\t'); k-- {
		lv[k] = para
	}
	var segs []vseg
	var lvls []uint8
	for i := ln.Start; i < ln.End; {
		for _, cl := range t.ctl[i] {
			if i > ln.Start {
				segs = append(segs, vseg{s: i, e: i, rtl: cl%2 == 1})
				lvls = append(lvls, cl)
			}
		}
		j := i + 1
		for j < ln.End && j < t.pend[i] && lv[j-ln.Start] == lv[i-ln.Start] && len(t.ctl[j]) == 0 {
			j++
		}
		w := t.toUnits(t.pos[j] - t.pos[i])
		if j == ln.End {
			w += ln.endAdj
		}
		segs = append(segs, vseg{s: i, e: j, rtl: lv[i-ln.Start]%2 == 1, w: w})
		lvls = append(lvls, lv[i-ln.Start])
		i = j
	}
	hi, lo := uint8(0), uint8(255)
	for _, l := range lvls {
		hi = max(hi, l)
		if l%2 == 1 {
			lo = min(lo, l)
		}
	}
	for l := hi; l >= lo && l > 0; l-- {
		for i := 0; i < len(segs); {
			if lvls[i] < l {
				i++
				continue
			}
			j := i
			for j < len(segs) && lvls[j] >= l {
				j++
			}
			for a, b := i, j-1; a < b; a, b = a+1, b-1 {
				segs[a], segs[b] = segs[b], segs[a]
				lvls[a], lvls[b] = lvls[b], lvls[a]
			}
			i = j
		}
	}
	x := ln.X
	for k := range segs {
		segs[k].x = x
		x += segs[k].w
	}
	ln.vis = segs
}

// visX returns the x of the logical boundary before rune i on a
// reordered line: the left edge of rune i in a left-to-right segment, its
// right edge in a right-to-left one. At the end of a segment (i == e) it
// is the edge after the segment's last rune.
func (t *Text) visX(ln *Line, i int) geom.Unit {
	for _, g := range ln.vis {
		if i < g.s || i > g.e || i == g.e && !(i == ln.End || g.e == t.nextSegStart(ln, i)) {
			continue
		}
		if g.rtl {
			return g.x + t.toUnitsFloor(t.pos[g.e]-t.pos[i])
		}
		return g.x + t.toUnitsFloor(t.pos[i]-t.pos[g.s])
	}
	return ln.X
}

// nextSegStart reports i itself when no visual segment starts at i (so a
// boundary at a segment end belongs to that segment), else -1.
func (t *Text) nextSegStart(ln *Line, i int) int {
	for _, g := range ln.vis {
		if g.s == i {
			return -1
		}
	}
	return i
}

// Spans returns the visual x ranges (relative to the content box) of
// runes [s, e) on line ln, left to right, adjacent ranges merged.
func (t *Text) Spans(ln *Line, s, e int) [][2]geom.Unit {
	if ln.vis == nil {
		return [][2]geom.Unit{{t.X(ln, s), t.X(ln, max(s, e))}}
	}
	var out [][2]geom.Unit
	for _, g := range ln.vis {
		a, b := max(s, g.s), min(e, g.e)
		if a >= b {
			continue
		}
		var r [2]geom.Unit
		if g.rtl {
			r = [2]geom.Unit{g.x + t.toUnitsFloor(t.pos[g.e]-t.pos[b]), g.x + t.toUnitsFloor(t.pos[g.e]-t.pos[a])}
			if b == g.e {
				r[0] = g.x
			}
			if a == g.s {
				r[1] = g.x + g.w
			}
		} else {
			r = [2]geom.Unit{g.x + t.toUnitsFloor(t.pos[a]-t.pos[g.s]), g.x + t.toUnitsFloor(t.pos[b]-t.pos[g.s])}
			if b == g.e {
				r[1] = g.x + g.w
			}
		}
		if n := len(out); n > 0 && out[n-1][1] == r[0] {
			out[n-1][1] = r[1]
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		x := t.X(ln, s)
		out = append(out, [2]geom.Unit{x, x})
	}
	return out
}

// visualGlyphs appends the glyphs of a reordered line, left to right.
func (t *Text) visualGlyphs(ln *Line, out []PlacedGlyph) []PlacedGlyph {
	for _, g := range ln.vis {
		g0 := 0
		for g0 < len(t.glyphs) && t.glyphs[g0].Cluster < g.s {
			g0++
		}
		g1 := g0
		for g1 < len(t.glyphs) && t.glyphs[g1].Cluster < g.e {
			g1++
		}
		var pen int64
		place := func(k int) {
			gl := t.glyphs[k]
			out = append(out, PlacedGlyph{
				ID: gl.ID, Face: gl.Face, Run: int(t.run[gl.Cluster]),
				X: g.x + t.toUnitsFloor(pen+gl.XOff),
				Y: ln.Baseline - t.toUnitsFloor(gl.YOff),
			})
			pen += gl.Advance
		}
		if g.rtl {
			for k := g1 - 1; k >= g0; k-- {
				place(k)
			}
		} else {
			for k := g0; k < g1; k++ {
				place(k)
			}
		}
	}
	return out
}

// Ink returns the horizontal extent of the line's characters other than
// white space, relative to the content box: in visual order, so white
// space that ends up at either edge of a reordered line is left out.
func (t *Text) Ink(ln *Line) (x0, x1 geom.Unit) {
	if ln.vis == nil {
		return ln.X, ln.X + t.InkWidth(ln)
	}
	first := true
	for i := ln.Start; i < ln.End; i++ {
		if t.runes[i] == ' ' || t.runes[i] == '\t' {
			continue
		}
		for _, sp := range t.Spans(ln, i, i+1) {
			if first || sp[0] < x0 {
				x0 = sp[0]
			}
			if first || sp[1] > x1 {
				x1 = sp[1]
			}
			first = false
		}
	}
	return x0, x1
}
