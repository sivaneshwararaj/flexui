package text

import (
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

func params(ws style.WhiteSpace) Params {
	return Params{Style: Style{Weight: 400, Size: geom.Px(16)}, WhiteSpace: ws}
}

// TestKerning guards the coverage workaround in coverage.go: class-based
// kerning pairs that go-text v0.3.5 skips must be applied. Widths are
// Chromium's (1/64 px) for Inter at 32px.
func TestKerning(t *testing.T) {
	for s, want := range map[string]geom.Unit{
		"To":    2390, // 37.34375px
		"fox":   2998, // 46.84375px
		"brown": 6068, // 94.8125px
		"AV":    2686, // 41.96875px
	} {
		p := params(style.WhiteSpaceNormal)
		p.Size = geom.Px(32)
		if got := New(s, p).MaxContent(); got != want {
			t.Errorf("%q: width %v, want %v", s, got, geom.Unit(want))
		}
	}
}

func TestWhiteSpace(t *testing.T) {
	for _, c := range []struct {
		ws       style.WhiteSpace
		in, want string
	}{
		{style.WhiteSpaceNormal, "  a \n\t b  ", "a b"},
		{style.WhiteSpaceNowrap, "a   b", "a b"},
		{style.WhiteSpacePre, "a  b\r\nc\n", "a  b\nc"},
		{style.WhiteSpacePreWrap, " a\rb ", " a\nb "},
		{style.WhiteSpaceNormal, " \n ", ""},
	} {
		if got := New(c.in, params(c.ws)).String(); got != c.want {
			t.Errorf("%v %q: got %q want %q", c.ws, c.in, got, c.want)
		}
	}
}

func TestWrapping(t *testing.T) {
	tx := New("aaa bbb ccc", params(style.WhiteSpaceNormal))
	minc, maxc := tx.MinContent(), tx.MaxContent()
	if minc <= 0 || maxc <= 2*minc {
		t.Fatalf("min %v max %v", minc, maxc)
	}
	lh := tx.LineHeight()
	if lh != geom.Px(20) { // Inter 16px: round(15.5) + round(3.86)
		t.Errorf("line height %v", lh)
	}
	for _, c := range []struct {
		w     geom.Unit
		lines int
	}{{maxc, 1}, {maxc - 1, 2}, {minc, 3}, {0, 3}} {
		if got := tx.Height(c.w); got != geom.Unit(c.lines)*lh {
			t.Errorf("width %v: height %v, want %d lines", c.w, got, c.lines)
		}
	}
	// Trailing spaces hang: the line width excludes them.
	b := tx.Layout(maxc - 1)
	if b.Lines[0].End != 7 || tx.Runes(b.Lines[0].Start, b.Lines[0].End) != "aaa bbb" {
		t.Errorf("line 0: %+v", b.Lines[0])
	}
	if n := New("a b c", params(style.WhiteSpaceNowrap)); n.Height(0) != lh || n.MinContent() != n.MaxContent() {
		t.Errorf("nowrap wrapped")
	}
	if n := New("a\nb\n\nc", params(style.WhiteSpacePre)); n.Height(0) != 4*lh {
		t.Errorf("pre lines: %v", n.Height(0)/lh)
	}
}

func TestEllipsisAndAlign(t *testing.T) {
	p := params(style.WhiteSpaceNowrap)
	p.Ellipsis = true
	tx := New("Hello wonderful world", p)
	w := tx.MaxContent() / 2
	ln := tx.Layout(w).Lines[0]
	if !ln.Ellipsis || ln.Width+tx.EllipsisWidth() > w || ln.End >= len([]rune(tx.String())) {
		t.Errorf("ellipsis line %+v", ln)
	}
	p = params(style.WhiteSpaceNormal)
	p.Align = style.TextAlignRight
	tx = New("Hi", p)
	if ln := tx.Layout(geom.Px(100)).Lines[0]; ln.X != geom.Px(100)-ln.Width {
		t.Errorf("right align x %v width %v", ln.X, ln.Width)
	}
}

func TestGlyphMaskDeterministic(t *testing.T) {
	f := ForWeight(400)
	id, _ := f.face.NominalGlyph('g')
	a := f.rasterize(uint16(id), geom.Px(17), 1)
	b := f.rasterize(uint16(id), geom.Px(17), 1)
	if a.W == 0 || a.H == 0 || string(a.Pix) != string(b.Pix) {
		t.Fatalf("mask %dx%d", a.W, a.H)
	}
	sum := 0
	for _, v := range a.Pix {
		sum += int(v)
	}
	// The glyph has some fully covered pixels and anti-aliased edges.
	full, partial := 0, 0
	for _, v := range a.Pix {
		if v == 255 {
			full++
		} else if v > 0 {
			partial++
		}
	}
	if full == 0 || partial == 0 || sum == 0 {
		t.Errorf("mask coverage: full %d partial %d", full, partial)
	}
}
