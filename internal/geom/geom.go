// Package geom provides the fixed-point length type and simple geometry
// shared by layout and paint.
//
// All layout math is integer math on Unit (1/64 of a logical pixel), which
// makes results bit-identical on every OS and architecture. Go is allowed to
// fuse float multiply-adds on some architectures (arm64, ppc64, s390x), so
// floats are avoided in layout entirely.
package geom

import (
	"math/bits"
	"strconv"
)

// Unit is a length in 1/64 logical pixels.
type Unit int64

// UnitsPerPx is the number of Units in one logical pixel.
const UnitsPerPx = 64

// Px converts whole pixels to Units.
func Px(v int) Unit { return Unit(v) * UnitsPerPx }

// Float returns u in pixels. Intended for output/debugging only; never feed
// the result back into layout math.
func (u Unit) Float() float64 { return float64(u) / UnitsPerPx }

// String formats u as an exact decimal pixel value (1/64 has a finite
// decimal expansion), e.g. "12.5px".
func (u Unit) String() string {
	return strconv.FormatFloat(u.Float(), 'f', -1, 64) + "px"
}

// Min returns the smaller of a and b.
func Min(a, b Unit) Unit {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max(a, b Unit) Unit {
	if a > b {
		return a
	}
	return b
}

// MulDiv returns a*b/c truncated toward zero, computed with a 128-bit
// intermediate so that it cannot overflow for any int64 inputs whose
// result fits. c must be non-zero. b and c must be non-negative.
func MulDiv(a, b, c int64) int64 {
	neg := a < 0
	ua := uint64(a)
	if neg {
		ua = uint64(-a)
	}
	hi, lo := bits.Mul64(ua, uint64(b))
	if hi >= uint64(c) {
		// Result would overflow; saturate.
		if neg {
			return -1 << 63
		}
		return 1<<63 - 1
	}
	q, _ := bits.Div64(hi, lo, uint64(c))
	if neg {
		return -int64(q)
	}
	return int64(q)
}

// Size is a width and height.
type Size struct{ W, H Unit }

// Point is a position.
type Point struct{ X, Y Unit }

// Rect is an axis-aligned rectangle with origin and size.
type Rect struct{ X, Y, W, H Unit }

// Edges holds per-side values (margins, borders, padding).
type Edges struct{ Top, Right, Bottom, Left Unit }

// Horizontal returns Left+Right.
func (e Edges) Horizontal() Unit { return e.Left + e.Right }

// Vertical returns Top+Bottom.
func (e Edges) Vertical() Unit { return e.Top + e.Bottom }
