package text

import (
	"bytes"
	_ "embed"
	"sort"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
)

//go:embed fonts/Inter-Regular.ttf
var interRegular []byte

//go:embed fonts/Inter-Bold.ttf
var interBold []byte

// Face is one bundled font face. All metrics are in font units (integers);
// conversion to pixels happens in this package with integer math.
type Face struct {
	name string
	face *font.Face
	hb   *harfbuzz.Font
	Upem int32
	// Ascent, Descent (positive, below the baseline), LineGap in font units.
	Ascent, Descent, LineGap int32

	mu    sync.Mutex
	buf   *harfbuzz.Buffer
	masks map[maskKey]*paint.GlyphMask
}

// Name returns the face's name (e.g. "Inter-Regular").
func (f *Face) Name() string { return f.name }

// Pack is a set of fallback faces registered by an optional font package
// (github.com/sivaneshwararaj/flexui/fonts/...). Priority orders the
// fallback chain (lower first), independently of import order. Bold may
// be nil: bold text then uses the regular face (no synthetic bold).
type Pack struct {
	Name          string
	Priority      int
	Regular, Bold []byte

	once          sync.Once
	regular, bold *Face
}

var (
	loadOnce sync.Once
	regular  *Face
	bold     *Face

	packMu sync.Mutex
	packs  []*Pack
	frozen bool
)

// Register adds a fallback font pack. It must be called before any text
// is laid out (font packages call it from init); later calls panic, so
// that layout never changes under a running program.
func Register(p *Pack) {
	packMu.Lock()
	defer packMu.Unlock()
	if frozen {
		panic("text: font pack " + p.Name + " registered after first use")
	}
	for _, q := range packs {
		if q.Name == p.Name {
			return
		}
	}
	packs = append(packs, p)
	sort.SliceStable(packs, func(i, j int) bool { return packs[i].Priority < packs[j].Priority })
}

// Packs returns the names of the registered packs in fallback order.
func Packs() []string {
	packMu.Lock()
	defer packMu.Unlock()
	var n []string
	for _, p := range packs {
		n = append(n, p.Name)
	}
	return n
}

func load(name string, data []byte) *Face {
	f, err := font.ParseTTF(bytes.NewReader(data))
	if err != nil {
		panic("text: bundled font " + name + ": " + err.Error())
	}
	fixCoverages(f.Font)
	hb := harfbuzz.NewFont(f)
	upem := int32(f.Upem())
	// Shape in font units: positions come out as exact integers.
	hb.XScale, hb.YScale = upem, upem
	ext, _ := f.FontHExtents()
	return &Face{
		name: name, face: f, hb: hb, Upem: upem,
		Ascent: int32(ext.Ascender), Descent: -int32(ext.Descender), LineGap: int32(ext.LineGap),
		buf: harfbuzz.NewBuffer(), masks: map[maskKey]*paint.GlyphMask{},
	}
}

func (p *Pack) load() {
	p.once.Do(func() {
		p.regular = load(p.Name+"-Regular", p.Regular)
		p.bold = p.regular
		if p.Bold != nil {
			p.bold = load(p.Name+"-Bold", p.Bold)
		}
	})
}

// setup loads Inter and freezes the pack list.
func setup() {
	loadOnce.Do(func() {
		regular = load("Inter-Regular", interRegular)
		bold = load("Inter-Bold", interBold)
		packMu.Lock()
		frozen = true
		packMu.Unlock()
	})
}

// Scaling font units to pixels, as Chromium does on Linux (FreeType for
// advances, HarfBuzz for GPOS adjustments), in 16.16 fixed point
// (1/65536 px), so that sums and their rounding match exactly:
//
//   - FreeType: x_scale = FT_DivFix(size in 26.6, upem), and the unhinted
//     advance linearHoriAdvance = FT_MulDiv(advance, x_scale, 64);
//   - HarfBuzz (font scale = size in 16.16): mult = (scale << 16) / upem,
//     and a value v scales to (v*mult + 32768) >> 16.
//
// size is the font size in 1/64 px (= 26.6).

func (f *Face) ftAdvance(v int64, size geom.Unit) int64 {
	u := int64(f.Upem)
	xs := (int64(size)<<16 + u/2) / u
	return (v*xs + 32) / 64
}

func (f *Face) hbScale(v int64, size geom.Unit) int64 {
	mult := (int64(size) * 1024 << 16) / int64(f.Upem)
	return (v*mult + 32768) >> 16
}

// ForWeight returns the primary face for a CSS font-weight: Inter Bold for
// weights above 500, Inter Regular otherwise (CSS font matching with faces
// at 400 and 700).
func ForWeight(weight int) *Face {
	setup()
	if weight > 500 {
		return bold
	}
	return regular
}

// Chain returns the fallback chain for a weight: the primary face, then
// each registered pack's face of that weight, in priority order.
func Chain(weight int) []*Face {
	c := []*Face{ForWeight(weight)}
	for _, p := range packs { // frozen: no lock needed
		p.load()
		if weight > 500 {
			c = append(c, p.bold)
		} else {
			c = append(c, p.regular)
		}
	}
	return c
}

// Glyph is one shaped glyph; advance and offsets are in 1/65536 px.
type Glyph struct {
	ID      uint16
	Face    *Face
	Cluster int // index of the first rune of its cluster
	Advance int64
	XOff    int64
	YOff    int64
}

// shapeProps are the HarfBuzz segment properties of an item.
type shapeProps struct {
	script language.Script
	lang   language.Language
	rtl    bool
}

// shape shapes runes [s, e) of text (the rest is context for shaping
// across item boundaries) left to right and appends the glyphs, with
// clusters as indices into text.
//
// Right-to-left text comes out of HarfBuzz in visual order; it is
// returned reversed, in logical (cluster) order like left-to-right text.
// Reversing a right-to-left range again gives the visual order.
func (f *Face) shape(text []rune, s, e int, p shapeProps, size geom.Unit, out []Glyph) []Glyph {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := len(out)
	b := f.buf
	b.Clear()
	b.AddRunes(text, s, e-s)
	b.Props.Direction = harfbuzz.LeftToRight
	if p.rtl {
		b.Props.Direction = harfbuzz.RightToLeft
	}
	b.Props.Script = p.script
	b.Props.Language = p.lang
	b.Shape(f.hb, nil)
	for i := range b.Info {
		q := b.Pos[i]
		id := b.Info[i].Glyph
		// The advance is the font's (FreeType) plus GPOS adjustments
		// (HarfBuzz); zeroed advances (marks) stay zero.
		var adv int64
		if q.XAdvance != 0 {
			base := int64(f.face.HorizontalAdvance(id))
			adv = f.ftAdvance(base, size) + f.hbScale(int64(q.XAdvance)-base, size)
		}
		out = append(out, Glyph{
			ID: uint16(id), Face: f, Cluster: b.Info[i].Cluster,
			Advance: adv, XOff: f.hbScale(int64(q.XOffset), size), YOff: f.hbScale(int64(q.YOffset), size),
		})
	}
	if p.rtl {
		g := out[n:]
		for i, j := 0, len(g)-1; i < j; i, j = i+1, j-1 {
			g[i], g[j] = g[j], g[i]
		}
	}
	return out
}
