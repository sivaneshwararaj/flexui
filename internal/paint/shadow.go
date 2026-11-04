package paint

import (
	"image"
	"math/bits"
)

// maskOne is the mask value for full coverage. Masks are kept at high
// precision (64 coverage steps x 1020) so the blur passes do not band.
const maskOne = 64 * 1020

// shadowGeometry returns the snapped shadow shape, the box-blur radius
// (device pixels) and the mask bounds (the shape grown by three blur
// radii), which bound every pixel the shadow can touch.
func (r *Raster) shadowGeometry(box RRect, s Shadow) (sh shape, rb int, mb image.Rectangle, ok bool) {
	sr := box
	sr.Rect.X += s.DX - s.Spread
	sr.Rect.Y += s.DY - s.Spread
	sr.Rect.W += 2 * s.Spread
	sr.Rect.H += 2 * s.Spread
	for i := range sr.Radii {
		if sr.Radii[i].X > 0 && sr.Radii[i].Y > 0 {
			sr.Radii[i].X = max(0, sr.Radii[i].X+s.Spread)
			sr.Radii[i].Y = max(0, sr.Radii[i].Y+s.Spread)
		}
	}
	sh = r.shapeOf(sr)
	if sh.empty() {
		return sh, 0, image.Rectangle{}, false
	}
	// Box-blur radius: three passes of width w = 2*rb+1 have variance
	// 3*(w*w-1)/12 = sigma², so w = sqrt(4*sigma²+1). sigma is in device
	// units (1/64 px).
	sigma := r.dev(s.Blur) / 2
	if sigma > 0 {
		wq := int64(isqrt(uint64(4*sigma*sigma + 64*64))) // width in 1/64 px
		w := int((wq + 32) / 64)
		if w%2 == 0 {
			w++
		}
		rb = w / 2
	}
	return sh, rb, sh.bounds().Inset(-3 * rb), true
}

// BoxShadow implements Canvas.
//
// The shadow shape is box offset by (DX, DY) and grown by Spread (radii
// grow by Spread too, never below 0). Its coverage mask is blurred with
// three box-blur passes per axis, approximating a Gaussian with
// sigma = Blur/2 as CSS specifies, then composited outside the box
// itself.
func (r *Raster) BoxShadow(box RRect, s Shadow) {
	if s.Color.A == 0 {
		return
	}
	sh, rb, mb, ok := r.shadowGeometry(box, s)
	if !ok {
		return
	}
	boxShape := r.shapeOf(box)
	area := mb.Intersect(r.clip().bounds)
	if area.Empty() {
		return
	}
	mw, mh := mb.Dx(), mb.Dy()
	if cap(r.mask) < mw*mh {
		r.mask = make([]uint32, mw*mh)
	}
	m := r.mask[:mw*mh]
	clear(m)
	sb := sh.bounds()
	for y := sb.Min.Y; y < sb.Max.Y; y++ {
		row := (y - mb.Min.Y) * mw
		for x := sb.Min.X; x < sb.Max.X; x++ {
			m[row+x-mb.Min.X] = uint32(sh.cov(x, y)) * 1020
		}
	}
	if rb > 0 {
		r.blur(m, mw, mh, rb)
	}
	rounded := r.hasRoundedClip()
	pix := r.dst.Pix
	c := s.Color
	for y := area.Min.Y; y < area.Max.Y; y++ {
		i := r.dst.PixOffset(area.Min.X, y)
		row := (y - mb.Min.Y) * mw
		for x := area.Min.X; x < area.Max.X; x, i = x+1, i+4 {
			v := uint64(m[row+x-mb.Min.X])
			if v == 0 {
				continue
			}
			// Not drawn under the box itself.
			k := uint64(64 - boxShape.cov(x, y))
			if rounded {
				k = k * uint64(r.clipCov(x, y)) >> 6
			}
			a := (uint64(c.A)*v*k + maskOne*32) / (maskOne * 64)
			blend(pix, i, c, uint32(a))
		}
	}
}

// blur applies three box-blur passes of radius rb horizontally, then
// three vertically, to the mw x mh mask m (in place). Values outside the
// mask are 0. Two reused scratch buffers are alternated.
func (r *Raster) blur(m []uint32, mw, mh, rb int) {
	n := mw * mh
	if cap(r.mask2) < n {
		r.mask2 = make([]uint32, n)
	}
	if cap(r.colsum) < mw {
		r.colsum = make([]uint64, mw)
	}
	src, dst := m, r.mask2[:n]
	w := uint64(2*rb + 1)
	half := w / 2
	// Exact division by the odd width w: floor(x/w) = hi64(x*magic) for
	// x < 2^32 (Granlund-Montgomery); integer-only and deterministic.
	magic := ^uint64(0)/w + 1
	div := func(x uint64) uint32 { hi, _ := bits.Mul64(x+half, magic); return uint32(hi) }
	for pass := 0; pass < 3; pass++ {
		for y := 0; y < mh; y++ {
			s, d := src[y*mw:(y+1)*mw], dst[y*mw:(y+1)*mw]
			var sum uint64
			for x := 0; x <= rb && x < mw; x++ {
				sum += uint64(s[x])
			}
			for x := 0; x < mw; x++ {
				d[x] = div(sum)
				if j := x + rb + 1; j < mw {
					sum += uint64(s[j])
				}
				if j := x - rb; j >= 0 {
					sum -= uint64(s[j])
				}
			}
		}
		src, dst = dst, src
	}
	cs := r.colsum[:mw]
	for pass := 0; pass < 3; pass++ {
		clear(cs)
		for y := 0; y <= rb && y < mh; y++ {
			row := src[y*mw : (y+1)*mw]
			for x, v := range row {
				cs[x] += uint64(v)
			}
		}
		for y := 0; y < mh; y++ {
			d := dst[y*mw : (y+1)*mw]
			for x := range d {
				d[x] = div(cs[x])
			}
			if j := y + rb + 1; j < mh {
				for x, v := range src[j*mw : (j+1)*mw] {
					cs[x] += uint64(v)
				}
			}
			if j := y - rb; j >= 0 {
				for x, v := range src[j*mw : (j+1)*mw] {
					cs[x] -= uint64(v)
				}
			}
		}
		src, dst = dst, src
	}
	// Six passes: the result is back in m.
}
