package paint

import (
	"image"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Image drawing (spec/SUBSET.md §6.5): the destination rectangle is
// snapped like any rectangle (§6.1 rule 2); each device pixel samples the
// source at its center with bilinear filtering in 16.16 fixed point on
// premultiplied RGBA (edges clamp), then composites source-over through
// the clip. A 1:1 mapping copies pixels exactly.

// DrawImage implements Canvas.
func (r *Raster) DrawImage(img *image.RGBA, dst geom.Rect) {
	if img == nil || img.Rect.Empty() {
		return
	}
	s := r.shapeOf(RRect{Rect: dst})
	dw, dh := s.x1-s.x0, s.y1-s.y0
	if dw <= 0 || dh <= 0 {
		return
	}
	sw, sh := img.Rect.Dx(), img.Rect.Dy()
	area := image.Rect(s.x0, s.y0, s.x1, s.y1).Intersect(r.clip().bounds)
	rounded := r.hasRoundedClip()
	pix := r.dst.Pix
	exact := dw == sw && dh == sh
	for py := area.Min.Y; py < area.Max.Y; py++ {
		i := r.dst.PixOffset(area.Min.X, py)
		// Source y of the pixel center, 16.16, minus half a texel.
		fy := int64(2*(py-s.y0)+1)*int64(sh)*32768/int64(dh) - 32768
		for px := area.Min.X; px < area.Max.X; px, i = px+1, i+4 {
			var c [4]uint32
			if exact {
				o := img.PixOffset(img.Rect.Min.X+px-s.x0, img.Rect.Min.Y+py-s.y0)
				c = [4]uint32{uint32(img.Pix[o]), uint32(img.Pix[o+1]), uint32(img.Pix[o+2]), uint32(img.Pix[o+3])}
			} else {
				fx := int64(2*(px-s.x0)+1)*int64(sw)*32768/int64(dw) - 32768
				c = bilinear(img, fx, fy)
			}
			if rounded {
				k := uint32(r.clipCov(px, py))
				for j := range c {
					c[j] = (c[j]*k + 32) >> 6
				}
			}
			blendPremul(pix, i, c)
		}
	}
}

// bilinear samples img at (fx, fy) (16.16 source pixel coordinates of the
// texel centers' grid) and returns premultiplied RGBA, rounded.
func bilinear(img *image.RGBA, fx, fy int64) [4]uint32 {
	clampi := func(v, hi int64) int64 { return min(max(v, 0), hi) }
	w, h := int64(img.Rect.Dx()), int64(img.Rect.Dy())
	x0, y0 := fx>>16, fy>>16
	ax, ay := uint64(fx&0xffff), uint64(fy&0xffff)
	xa, xb := clampi(x0, w-1), clampi(x0+1, w-1)
	ya, yb := clampi(y0, h-1), clampi(y0+1, h-1)
	at := func(x, y int64) int { return img.PixOffset(img.Rect.Min.X+int(x), img.Rect.Min.Y+int(y)) }
	p00, p10, p01, p11 := at(xa, ya), at(xb, ya), at(xa, yb), at(xb, yb)
	var out [4]uint32
	for k := 0; k < 4; k++ {
		top := uint64(img.Pix[p00+k])*(65536-ax) + uint64(img.Pix[p10+k])*ax
		bot := uint64(img.Pix[p01+k])*(65536-ax) + uint64(img.Pix[p11+k])*ax
		v := top*(65536-ay) + bot*ay
		out[k] = uint32((v + 1<<31) >> 32)
	}
	// Premultiplied invariants: channels never exceed alpha.
	for k := 0; k < 3; k++ {
		out[k] = min(out[k], out[3])
	}
	return out
}

// blendPremul composites a premultiplied source pixel over the pixel at
// offset i: dst = src + dst*(255-a)/255 (rounded, §6.3).
func blendPremul(pix []uint8, i int, c [4]uint32) {
	a := c[3]
	if a == 0 {
		return
	}
	inv := 255 - a
	for k := 0; k < 4; k++ {
		pix[i+k] = uint8(min(255, c[k]+div255(uint32(pix[i+k])*inv)))
	}
}
