package text

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"

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

var (
	loadOnce sync.Once
	regular  *Face
	bold     *Face
)

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

// ForWeight returns the bundled face for a CSS font-weight: Inter Bold for
// weights above 500, Inter Regular otherwise (CSS font matching with faces
// at 400 and 700).
func ForWeight(weight int) *Face {
	loadOnce.Do(func() {
		regular = load("Inter-Regular", interRegular)
		bold = load("Inter-Bold", interBold)
	})
	if weight > 500 {
		return bold
	}
	return regular
}

// Glyph is one shaped glyph, positioned in font units relative to the
// start of the shaped text.
type Glyph struct {
	ID      uint16
	Cluster int   // index of the first rune of its cluster
	Advance int32 // font units
	XOff    int32
	YOff    int32
}

// shape shapes runes left to right (Latin script) and appends the glyphs.
func (f *Face) shape(runes []rune, out []Glyph) []Glyph {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.buf
	b.Clear()
	b.AddRunes(runes, 0, len(runes))
	b.GuessSegmentProperties()
	b.Props.Language = language.NewLanguage("en")
	b.Shape(f.hb, nil)
	for i := range b.Info {
		p := b.Pos[i]
		out = append(out, Glyph{
			ID: uint16(b.Info[i].Glyph), Cluster: b.Info[i].Cluster,
			Advance: p.XAdvance, XOff: p.XOffset, YOff: p.YOffset,
		})
	}
	return out
}
