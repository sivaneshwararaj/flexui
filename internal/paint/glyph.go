package paint

import (
	"image"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// GlyphMask is a rasterized glyph: 8-bit coverage (0..255) of W x H
// pixels whose top-left pixel is at (X, Y) relative to the pixel that
// contains the glyph origin.
type GlyphMask struct {
	X, Y, W, H int
	Pix        []uint8
}

// GlyphSource rasterizes glyphs. It is implemented by the text package's
// font faces, so that paint does not depend on fonts. Masks must be
// deterministic and may be cached; callers must not modify them.
type GlyphSource interface {
	// GlyphMask returns the mask of glyph id at a pixel size of ppem
	// (device pixels per em, in 1/64), with the origin subX/4 of a pixel
	// to the right of the pixel's left edge (subX in 0..3).
	GlyphMask(id uint16, ppem geom.Unit, subX int) *GlyphMask
}

// Glyph is a glyph id with the position of its origin (pen position on
// the baseline) in logical units.
type Glyph struct {
	ID   uint16
	X, Y geom.Unit
}

// GlyphRun is a run of glyphs of one font, size and color.
type GlyphRun struct {
	Font   GlyphSource
	Size   geom.Unit // font size in logical units (1/64 px)
	Glyphs []Glyph
	Color  Color
}

// Glyph snapping (spec/SUBSET.md §6.1, rule 5): the baseline is rounded
// to the nearest device pixel and the pen x to the nearest quarter of a
// device pixel; the size scales exactly with the device scale.

// Glyphs implements Canvas.
func (r *Raster) Glyphs(run *GlyphRun) {
	if run.Color.A == 0 || len(run.Glyphs) == 0 {
		return
	}
	ppem := geom.Unit(r.dev(run.Size))
	bounds := r.clip().bounds
	rounded := r.hasRoundedClip()
	pix := r.dst.Pix
	c := run.Color
	for _, g := range run.Glyphs {
		q := (r.dev(g.X) + 8) >> 4 // quarter pixels
		ox, sub := int(q>>2), int(q&3)
		oy := snap(r.dev(g.Y))
		m := run.Font.GlyphMask(g.ID, ppem, sub)
		if m == nil || m.W == 0 {
			continue
		}
		area := image.Rect(ox+m.X, oy+m.Y, ox+m.X+m.W, oy+m.Y+m.H).Intersect(bounds)
		for py := area.Min.Y; py < area.Max.Y; py++ {
			mi := (py-oy-m.Y)*m.W + area.Min.X - ox - m.X
			i := r.dst.PixOffset(area.Min.X, py)
			for px := area.Min.X; px < area.Max.X; px, i, mi = px+1, i+4, mi+1 {
				k := uint32(m.Pix[mi])
				if k == 0 {
					continue
				}
				a := div255(uint32(c.A) * k)
				if rounded {
					a = (a*uint32(r.clipCov(px, py)) + 32) >> 6
				}
				blend(pix, i, c, a)
			}
		}
	}
}
