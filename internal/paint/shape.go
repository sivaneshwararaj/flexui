package paint

import (
	"image"
	"math/bits"
)

// Device-space geometry. Device units are 1/64 device pixel; rectangle
// edges are whole device pixels after snapping.

// rad is a corner radius in device units.
type rad struct{ x, y int64 }

// shape is a snapped rounded rectangle in device space.
type shape struct {
	x0, y0, x1, y1 int // pixel edges, x0 <= x1, y0 <= y1
	r              [4]rad
	rounded        bool
	// Corner zone sizes in whole pixels (pixels that a corner touches).
	zx, zy [4]int
}

func (s *shape) empty() bool { return s.x0 >= s.x1 || s.y0 >= s.y1 }

func (s *shape) bounds() image.Rectangle { return image.Rect(s.x0, s.y0, s.x1, s.y1) }

// normalize scales radii down uniformly when adjacent radii exceed a side
// (CSS Backgrounds §5.5) and prepares corner zones.
func (s *shape) normalize() {
	w := int64(s.x1-s.x0) * 64
	h := int64(s.y1-s.y0) * 64
	for i := range s.r {
		if s.r[i].x <= 0 || s.r[i].y <= 0 {
			s.r[i] = rad{}
		}
	}
	// Smallest ratio side/sum over the four sides, as a fraction num/den.
	num, den := int64(1), int64(1)
	check := func(side, sum int64) {
		if sum > 0 && side*den < num*sum {
			num, den = side, sum
		}
	}
	check(w, s.r[TopLeft].x+s.r[TopRight].x)
	check(w, s.r[BottomLeft].x+s.r[BottomRight].x)
	check(h, s.r[TopLeft].y+s.r[BottomLeft].y)
	check(h, s.r[TopRight].y+s.r[BottomRight].y)
	if num < den {
		for i := range s.r {
			s.r[i].x = s.r[i].x * num / den
			s.r[i].y = s.r[i].y * num / den
		}
	}
	s.rounded = false
	for i := range s.r {
		if s.r[i].x > 0 && s.r[i].y > 0 {
			s.rounded = true
			s.zx[i] = int((s.r[i].x + 63) / 64)
			s.zy[i] = int((s.r[i].y + 63) / 64)
		} else {
			s.r[i] = rad{}
			s.zx[i], s.zy[i] = 0, 0
		}
	}
}

// cov returns the coverage of pixel (px, py) by s, in 0..64 (8x8 samples).
// Straight edges are pixel-aligned, so only corner pixels are partial.
func (s *shape) cov(px, py int) int {
	if px < s.x0 || px >= s.x1 || py < s.y0 || py >= s.y1 {
		return 0
	}
	if !s.rounded || !s.inCornerZone(px, py) {
		return 64
	}
	if c, ok := s.quick(px, py); ok {
		return c
	}
	n := 0
	for j := 0; j < 8; j++ {
		sy := int64(py)*64 + 4 + int64(j)*8
		for i := 0; i < 8; i++ {
			sx := int64(px)*64 + 4 + int64(i)*8
			if s.insideSample(sx, sy) {
				n++
			}
		}
	}
	return n
}

// quick classifies a pixel that lies entirely inside one corner's
// quadrant. There the inside test is monotonic in the distance from the
// corner center along each axis, so the pixel corner nearest the center
// decides "fully outside" and the farthest one "fully inside". The result
// equals what supersampling would give (samples lie between the two).
func (s *shape) quick(px, py int) (int, bool) {
	X0, Y0, X1, Y1 := int64(s.x0)*64, int64(s.y0)*64, int64(s.x1)*64, int64(s.y1)*64
	l, t := int64(px)*64, int64(py)*64
	rgt, bot := l+64, t+64
	var near, far [2]int64
	var r rad
	switch {
	case s.r[TopLeft].x > 0 && rgt <= X0+s.r[TopLeft].x && bot <= Y0+s.r[TopLeft].y:
		r = s.r[TopLeft]
		cx, cy := X0+r.x, Y0+r.y
		near, far = [2]int64{cx - rgt, cy - bot}, [2]int64{cx - l, cy - t}
	case s.r[TopRight].x > 0 && l >= X1-s.r[TopRight].x && bot <= Y0+s.r[TopRight].y:
		r = s.r[TopRight]
		cx, cy := X1-r.x, Y0+r.y
		near, far = [2]int64{l - cx, cy - bot}, [2]int64{rgt - cx, cy - t}
	case s.r[BottomRight].x > 0 && l >= X1-s.r[BottomRight].x && t >= Y1-s.r[BottomRight].y:
		r = s.r[BottomRight]
		cx, cy := X1-r.x, Y1-r.y
		near, far = [2]int64{l - cx, t - cy}, [2]int64{rgt - cx, bot - cy}
	case s.r[BottomLeft].x > 0 && rgt <= X0+s.r[BottomLeft].x && t >= Y1-s.r[BottomLeft].y:
		r = s.r[BottomLeft]
		cx, cy := X0+r.x, Y1-r.y
		near, far = [2]int64{cx - rgt, t - cy}, [2]int64{cx - l, bot - cy}
	default:
		return 0, false
	}
	if !inEllipse(near[0], near[1], r) {
		return 0, true
	}
	if inEllipse(far[0], far[1], r) {
		return 64, true
	}
	return 0, false
}

func (s *shape) inCornerZone(px, py int) bool {
	return (px < s.x0+s.zx[TopLeft] && py < s.y0+s.zy[TopLeft]) ||
		(px >= s.x1-s.zx[TopRight] && py < s.y0+s.zy[TopRight]) ||
		(px >= s.x1-s.zx[BottomRight] && py >= s.y1-s.zy[BottomRight]) ||
		(px < s.x0+s.zx[BottomLeft] && py >= s.y1-s.zy[BottomLeft])
}

// insideSample tests a sample point (device units) known to be inside the
// rectangle against the corner ellipses.
func (s *shape) insideSample(sx, sy int64) bool {
	X0, Y0, X1, Y1 := int64(s.x0)*64, int64(s.y0)*64, int64(s.x1)*64, int64(s.y1)*64
	if r := s.r[TopLeft]; r.x > 0 {
		if cx, cy := X0+r.x, Y0+r.y; sx < cx && sy < cy {
			return inEllipse(cx-sx, cy-sy, r)
		}
	}
	if r := s.r[TopRight]; r.x > 0 {
		if cx, cy := X1-r.x, Y0+r.y; sx > cx && sy < cy {
			return inEllipse(sx-cx, cy-sy, r)
		}
	}
	if r := s.r[BottomRight]; r.x > 0 {
		if cx, cy := X1-r.x, Y1-r.y; sx > cx && sy > cy {
			return inEllipse(sx-cx, sy-cy, r)
		}
	}
	if r := s.r[BottomLeft]; r.x > 0 {
		if cx, cy := X0+r.x, Y1-r.y; sx < cx && sy > cy {
			return inEllipse(cx-sx, sy-cy, r)
		}
	}
	return true
}

// inEllipse reports dx²/rx² + dy²/ry² <= 1, i.e.
// (dx*ry)² + (dy*rx)² <= (rx*ry)², with 128-bit intermediates.
func inEllipse(dx, dy int64, r rad) bool {
	a := uint64(dx) * uint64(r.y)
	b := uint64(dy) * uint64(r.x)
	c := uint64(r.x) * uint64(r.y)
	ah, al := bits.Mul64(a, a)
	bh, bl := bits.Mul64(b, b)
	ch, cl := bits.Mul64(c, c)
	sl, carry := bits.Add64(al, bl, 0)
	sh, over := bits.Add64(ah, bh, carry)
	if over != 0 {
		return false
	}
	return sh < ch || (sh == ch && sl <= cl)
}

// isqrt returns floor(sqrt(v)).
func isqrt(v uint64) uint64 {
	if v == 0 {
		return 0
	}
	x := uint64(1) << ((bits.Len64(v) + 1) / 2)
	for {
		y := (x + v/x) / 2
		if y >= x {
			return x
		}
		x = y
	}
}
