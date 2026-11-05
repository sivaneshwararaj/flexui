package jpeg

import (
	"image"
	"io"
)

// DecodeRGBA decodes a JPEG image into a new *image.RGBA with bounds
// starting at (0, 0). The color conversions are integer math copied from
// Go 1.24's image/color (YCbCrToRGB, CMYKToRGB), so the result is the same
// on every platform and Go version.
func DecodeRGBA(r io.Reader) (*image.RGBA, error) {
	m, err := Decode(r)
	if err != nil {
		return nil, err
	}
	b := m.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	switch m := m.(type) {
	case *image.YCbCr:
		ycbcrInto(out, m)
	case *image.RGBA: // RGB (Adobe transform 0) JPEGs, opaque
		for y := 0; y < b.Dy(); y++ {
			copy(out.Pix[y*out.Stride:y*out.Stride+4*b.Dx()], m.Pix[m.PixOffset(b.Min.X, b.Min.Y+y):])
		}
	case *image.Gray:
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				v := m.Pix[m.PixOffset(b.Min.X+x, b.Min.Y+y)]
				i := out.PixOffset(x, y)
				out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = v, v, v, 0xff
			}
		}
	case *image.CMYK:
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				j := m.PixOffset(b.Min.X+x, b.Min.Y+y)
				r, g, bb := cmykToRGB(m.Pix[j], m.Pix[j+1], m.Pix[j+2], m.Pix[j+3])
				i := out.PixOffset(x, y)
				out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = r, g, bb, 0xff
			}
		}
	default:
		return nil, UnsupportedError("decoded image type")
	}
	return out, nil
}

// ycbcrInto converts src into dst (same size, both starting at their own
// Min), any chroma subsampling.
func ycbcrInto(dst *image.RGBA, src *image.YCbCr) {
	b := src.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		i := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y-b.Min.Y)
		for x := b.Min.X; x < b.Max.X; x, i = x+1, i+4 {
			r, g, bb := ycbcrToRGB(src.Y[src.YOffset(x, y)], src.Cb[src.COffset(x, y)], src.Cr[src.COffset(x, y)])
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = r, g, bb, 0xff
		}
	}
}

// ycbcrToRGB is Go 1.24's color.YCbCrToRGB (JFIF formulae in 16.16 fixed
// point, rounding adjustment 0x101*y/... as documented there).
func ycbcrToRGB(y, cb, cr uint8) (uint8, uint8, uint8) {
	yy1 := int32(y) * 0x10101
	cb1 := int32(cb) - 128
	cr1 := int32(cr) - 128
	return clamp16(yy1 + 91881*cr1), clamp16(yy1 - 22554*cb1 - 46802*cr1), clamp16(yy1 + 116130*cb1)
}

// clamp16 shifts a 16.16 value right by 16 and clamps it to [0, 255].
func clamp16(v int32) uint8 {
	if uint32(v)&0xff000000 == 0 {
		return uint8(v >> 16)
	}
	return uint8(^(v >> 31))
}

// cmykToRGB is Go 1.24's color.CMYKToRGB.
func cmykToRGB(c, m, y, k uint8) (uint8, uint8, uint8) {
	w := 0xffff - uint32(k)*0x101
	r := (0xffff - uint32(c)*0x101) * w / 0xffff
	g := (0xffff - uint32(m)*0x101) * w / 0xffff
	b := (0xffff - uint32(y)*0x101) * w / 0xffff
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}
