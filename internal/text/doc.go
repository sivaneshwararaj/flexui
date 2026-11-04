// Package text lays out and rasterizes text with the bundled Inter faces
// (spec/SUBSET.md §7): white-space processing, shaping (HarfBuzz, in font
// units), UAX #14 line breaking, greedy wrapping, intrinsic widths,
// alignment, ellipsis, and an integer-only glyph rasterizer that
// implements paint.GlyphSource.
//
// All widths are computed from integer font-unit advances and converted to
// geom.Unit with the rounding rules Chromium uses, so text layout matches
// Chromium exactly (see internal/render/text_test.go).
package text
