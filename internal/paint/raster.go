package paint

import (
	"image"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Raster is the software Canvas. It renders into an *image.RGBA
// (premultiplied alpha) using integer arithmetic only.
//
// Pixel-snapping rule (spec/SUBSET.md §6):
//
//  1. Logical coordinates (1/64 px) are scaled to device units (1/64
//     device px) by the scale factor, which is itself fixed-point
//     (Scale64 = 64 means 1.0).
//  2. Every rectangle edge (fills, borders' outer edges, clip rects,
//     shadow shapes after offset and spread) is rounded independently to
//     the nearest device-pixel boundary, halves rounding up:
//     x0 = round(x), x1 = round(x+w). Boxes that touch in layout therefore
//     touch exactly in pixels, with no gaps or double-painted seams; a
//     box's pixel size can differ from its layout size by one pixel.
//  3. Border widths are rounded to whole device pixels, with a minimum of
//     one pixel for any non-zero width. The inner edge is the snapped
//     outer edge minus the snapped width, so all four sides of a uniform
//     border are equally thick.
//  4. Corner radii, blur and spread are not snapped. Rounded corners are
//     anti-aliased with 8x8 supersampling; shadows are blurred.
//
// A Raster can be reused across frames; its buffers are kept.
type Raster struct {
	dst    *image.RGBA // current target (the frame or an opacity layer)
	frame  *image.RGBA
	scale  int64 // device units per logical unit, times 64
	clips  []clipEntry
	layers []layerEntry
	pool   []*image.RGBA
	mask   []uint32 // shadow mask scratch
	mask2  []uint32 // blur scratch
	colsum []uint64 // blur scratch
}

type clipEntry struct {
	bounds image.Rectangle // intersection of all clips so far
	shape  shape           // only used when rounded
}

type layerEntry struct {
	img    *image.RGBA
	prev   *image.RGBA
	bounds image.Rectangle
	alpha  uint32
}

// Scale64 converts a float scale factor (device pixels per logical pixel)
// to the rasterizer's fixed-point scale, rounding to the nearest 1/64.
// This is the only float operation, done once per frame at the platform
// boundary.
func Scale64(scale float64) int64 {
	if scale <= 0 {
		return 64
	}
	return int64(scale*64 + 0.5)
}

// Begin starts drawing a frame into dst at the given fixed-point scale
// (see Scale64). It resets the clip and layer stacks; dst is not cleared.
func (r *Raster) Begin(dst *image.RGBA, scale64 int64) {
	if scale64 <= 0 {
		scale64 = 64
	}
	r.dst, r.frame, r.scale = dst, dst, scale64
	r.clips = append(r.clips[:0], clipEntry{bounds: dst.Rect})
	r.layers = r.layers[:0]
}

// End finishes the frame, closing any groups left open.
func (r *Raster) End() {
	for len(r.layers) > 0 {
		r.PopOpacity()
	}
	r.clips = r.clips[:1]
}

// dev converts logical units to device units.
func (r *Raster) dev(u geom.Unit) int64 { return geom.MulDiv(int64(u), r.scale, 64) }

// snap rounds device units to the nearest pixel boundary (halves up).
func snap(d int64) int { return int((d + 32) >> 6) }

func (r *Raster) shapeOf(rr RRect) shape {
	x, y := r.dev(rr.Rect.X), r.dev(rr.Rect.Y)
	s := shape{
		x0: snap(x), y0: snap(y),
		x1: snap(x + r.dev(rr.Rect.W)), y1: snap(y + r.dev(rr.Rect.H)),
	}
	if s.x1 < s.x0 {
		s.x1 = s.x0
	}
	if s.y1 < s.y0 {
		s.y1 = s.y0
	}
	for i, rd := range rr.Radii {
		s.r[i] = rad{r.dev(rd.X), r.dev(rd.Y)}
	}
	s.normalize()
	return s
}

// snapWidth converts a border width to whole device pixels (rule 3).
func (r *Raster) snapWidth(w geom.Unit) int {
	if w <= 0 {
		return 0
	}
	px := snap(r.dev(w))
	if px < 1 {
		px = 1
	}
	return px
}

func (r *Raster) clip() *clipEntry { return &r.clips[len(r.clips)-1] }

// clipCov returns the clip coverage of a pixel inside the clip bounds,
// 0..64.
func (r *Raster) clipCov(px, py int) int {
	c := 64
	for i := 1; i < len(r.clips); i++ {
		if s := &r.clips[i].shape; s.rounded {
			c = c * s.cov(px, py) >> 6
			if c == 0 {
				return 0
			}
		}
	}
	return c
}

func (r *Raster) hasRoundedClip() bool {
	for i := 1; i < len(r.clips); i++ {
		if r.clips[i].shape.rounded {
			return true
		}
	}
	return false
}

// div255 returns round(x/255) for 0 <= x <= 255*255*... (exact for
// x < 65536*255).
func div255(x uint32) uint32 { return (x + 128 + ((x + 128) >> 8)) >> 8 }

// blend composites color c with effective alpha a (0..255, already
// including coverage) over the pixel at offset i, source-over.
func blend(pix []uint8, i int, c Color, a uint32) {
	if a == 0 {
		return
	}
	inv := 255 - a
	pix[i+0] = uint8(div255(uint32(c.R)*a + uint32(pix[i+0])*inv))
	pix[i+1] = uint8(div255(uint32(c.G)*a + uint32(pix[i+1])*inv))
	pix[i+2] = uint8(div255(uint32(c.B)*a + uint32(pix[i+2])*inv))
	pix[i+3] = uint8(a + div255(uint32(pix[i+3])*inv))
}

// paintCov fills the pixels of area with color c, each weighted by
// cov(px, py) (0..64) and by the current clip.
func (r *Raster) paintCov(area image.Rectangle, c Color, cov func(px, py int) int) {
	area = area.Intersect(r.clip().bounds)
	if area.Empty() || c.A == 0 {
		return
	}
	rounded := r.hasRoundedClip()
	pix := r.dst.Pix
	for py := area.Min.Y; py < area.Max.Y; py++ {
		i := r.dst.PixOffset(area.Min.X, py)
		for px := area.Min.X; px < area.Max.X; px, i = px+1, i+4 {
			k := cov(px, py)
			if k == 0 {
				continue
			}
			if rounded {
				k = k * r.clipCov(px, py) >> 6
			}
			blend(pix, i, c, (uint32(c.A)*uint32(k)+32)>>6)
		}
	}
}

// FillRRect implements Canvas.
func (r *Raster) FillRRect(rr RRect, c Color) {
	s := r.shapeOf(rr)
	if s.empty() || c.A == 0 {
		return
	}
	area := s.bounds().Intersect(r.clip().bounds)
	if area.Empty() {
		return
	}
	if r.hasRoundedClip() {
		r.paintCov(area, c, s.cov)
		return
	}
	// Rows that cross no corner zone are fully covered: fill the span
	// directly (opaque) or with a constant alpha. Same result as the
	// per-pixel path, much faster.
	top := s.y0 + max(s.zy[TopLeft], s.zy[TopRight])
	bot := s.y1 - max(s.zy[BottomLeft], s.zy[BottomRight])
	pix := r.dst.Pix
	a := uint32(c.A)
	for py := area.Min.Y; py < area.Max.Y; py++ {
		if s.rounded && (py < top || py >= bot) {
			r.paintCov(image.Rect(area.Min.X, py, area.Max.X, py+1), c, s.cov)
			continue
		}
		i := r.dst.PixOffset(area.Min.X, py)
		end := i + 4*area.Dx()
		if a == 255 {
			fillOpaque(pix[i:end], c)
		} else {
			for ; i < end; i += 4 {
				blend(pix, i, c, a)
			}
		}
	}
}

// Border implements Canvas.
func (r *Raster) Border(rr RRect, w geom.Edges, c Color) {
	outer := r.shapeOf(rr)
	if outer.empty() {
		return
	}
	wt, wr, wb, wl := r.snapWidth(w.Top), r.snapWidth(w.Right), r.snapWidth(w.Bottom), r.snapWidth(w.Left)
	if wt+wr+wb+wl == 0 {
		return
	}
	inner := shape{x0: outer.x0 + wl, y0: outer.y0 + wt, x1: outer.x1 - wr, y1: outer.y1 - wb}
	if inner.x1 < inner.x0 {
		inner.x1 = inner.x0
	}
	if inner.y1 < inner.y0 {
		inner.y1 = inner.y0
	}
	sub := func(a int64, px int) int64 { return max(0, a-int64(px)*64) }
	o := outer.r
	inner.r[TopLeft] = rad{sub(o[TopLeft].x, wl), sub(o[TopLeft].y, wt)}
	inner.r[TopRight] = rad{sub(o[TopRight].x, wr), sub(o[TopRight].y, wt)}
	inner.r[BottomRight] = rad{sub(o[BottomRight].x, wr), sub(o[BottomRight].y, wb)}
	inner.r[BottomLeft] = rad{sub(o[BottomLeft].x, wl), sub(o[BottomLeft].y, wb)}
	inner.normalize()
	cov := func(px, py int) int {
		return max(0, outer.cov(px, py)-inner.cov(px, py))
	}
	// Inside the inner shape, away from its corner zones, the inner
	// coverage is full, so the border's is 0: paint only the four bands
	// around that interior (disjoint, so the result is the same as
	// visiting every pixel of the box, just faster).
	mzx := max(inner.zx[0], inner.zx[1], inner.zx[2], inner.zx[3])
	mzy := max(inner.zy[0], inner.zy[1], inner.zy[2], inner.zy[3])
	sx0, sx1 := inner.x0+mzx, inner.x1-mzx
	sy0, sy1 := inner.y0+mzy, inner.y1-mzy
	b := outer.bounds()
	if sx0 >= sx1 || sy0 >= sy1 {
		r.paintCov(b, c, cov)
		return
	}
	r.paintCov(image.Rect(b.Min.X, b.Min.Y, b.Max.X, sy0), c, cov)
	r.paintCov(image.Rect(b.Min.X, sy1, b.Max.X, b.Max.Y), c, cov)
	r.paintCov(image.Rect(b.Min.X, sy0, sx0, sy1), c, cov)
	r.paintCov(image.Rect(sx1, sy0, b.Max.X, sy1), c, cov)
}

// PushClip implements Canvas.
func (r *Raster) PushClip(rr RRect) {
	s := r.shapeOf(rr)
	e := clipEntry{bounds: r.clip().bounds.Intersect(s.bounds()), shape: s}
	r.clips = append(r.clips, e)
}

// PopClip implements Canvas.
func (r *Raster) PopClip() {
	if len(r.clips) > 1 {
		r.clips = r.clips[:len(r.clips)-1]
	}
}

// PushOpacity implements Canvas. Drawing until PopOpacity goes into an
// offscreen layer (from a reused pool) covering the current clip bounds.
func (r *Raster) PushOpacity(alpha uint8) {
	b := r.clip().bounds
	var img *image.RGBA
	for i, p := range r.pool {
		if p.Rect == r.frame.Rect {
			img = p
			r.pool = append(r.pool[:i], r.pool[i+1:]...)
			break
		}
	}
	if img == nil {
		img = image.NewRGBA(r.frame.Rect)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		i := img.PixOffset(b.Min.X, y)
		clear(img.Pix[i : i+4*b.Dx()])
	}
	r.layers = append(r.layers, layerEntry{img: img, prev: r.dst, bounds: b, alpha: uint32(alpha)})
	r.dst = img
}

// PopOpacity implements Canvas.
func (r *Raster) PopOpacity() {
	if len(r.layers) == 0 {
		return
	}
	l := r.layers[len(r.layers)-1]
	r.layers = r.layers[:len(r.layers)-1]
	r.dst = l.prev
	src, dst := l.img.Pix, r.dst.Pix
	for y := l.bounds.Min.Y; y < l.bounds.Max.Y; y++ {
		i := l.img.PixOffset(l.bounds.Min.X, y)
		for x := l.bounds.Min.X; x < l.bounds.Max.X; x, i = x+1, i+4 {
			sa := div255(uint32(src[i+3]) * l.alpha)
			if sa == 0 {
				continue
			}
			inv := 255 - sa
			for k := 0; k < 3; k++ {
				dst[i+k] = uint8(div255(uint32(src[i+k])*l.alpha) + div255(uint32(dst[i+k])*inv))
			}
			dst[i+3] = uint8(sa + div255(uint32(dst[i+3])*inv))
		}
	}
	r.pool = append(r.pool, l.img)
}

// fillOpaque fills a span of RGBA pixels with an opaque color: one pixel
// is written, then copied in doubling chunks.
func fillOpaque(span []uint8, c Color) {
	if len(span) < 4 {
		return
	}
	span[0], span[1], span[2], span[3] = c.R, c.G, c.B, 255
	for n := 4; n < len(span); n *= 2 {
		copy(span[n:], span[:n])
	}
}
