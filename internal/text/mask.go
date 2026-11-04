package text

import (
	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
)

// Glyph rasterization (spec/SUBSET.md §7.4), integer-only:
//
//  1. Outline points are in font units; TrueType implied on-curve points
//     are midpoints, so coordinates are converted exactly to half font
//     units (the float32 values from the parser are exact).
//  2. Points are scaled to device space in 1/256 px (rounded), with the
//     origin offset by subX/4 px horizontally.
//  3. Quadratic and cubic curves are flattened into line segments by
//     uniform subdivision with an integer-computed step count that keeps
//     the flattening error under about 1/16 px.
//  4. Coverage: 16 sample rows per pixel (at row + (k+0.5)/16); on each,
//     the nonzero winding spans are intersected with the pixel columns
//     exactly in x (1/256 px). Coverage = covered length summed over the
//     rows, scaled to 0..255 with rounding.

const (
	fracBits = 8 // 1/256 px
	one      = 1 << fracBits
	subRows  = 16
	maxCache = 1 << 15
)

type maskKey struct {
	id   uint16
	ppem geom.Unit
	subX uint8
}

// GlyphMask implements paint.GlyphSource. Masks are cached per face.
func (f *Face) GlyphMask(id uint16, ppem geom.Unit, subX int) *paint.GlyphMask {
	k := maskKey{id, ppem, uint8(subX)}
	f.mu.Lock()
	m, ok := f.masks[k]
	f.mu.Unlock()
	if ok {
		return m
	}
	m = f.rasterize(id, ppem, subX)
	f.mu.Lock()
	if len(f.masks) >= maxCache {
		clear(f.masks)
	}
	f.masks[k] = m
	f.mu.Unlock()
	return m
}

type pt struct{ x, y int64 } // device space, 1/256 px, y down

type edge struct {
	x0, y0, x1, y1 int64 // y0 < y1
	dir            int   // +1 downward in the outline, -1 upward
}

func (f *Face) rasterize(id uint16, ppem geom.Unit, subX int) *paint.GlyphMask {
	out, ok := f.face.GlyphDataOutline(font.GID(id))
	if !ok || len(out.Segments) == 0 {
		return &paint.GlyphMask{}
	}
	// Half font units -> 1/256 px: v * ppem/64 * 256 / (2*upem).
	den := int64(2 * f.Upem)
	num := int64(ppem) * 4
	conv := func(p ot.SegmentPoint) pt {
		hx, hy := int64(2*p.X), int64(2*p.Y) // exact
		return pt{roundDiv(hx*num, den) + int64(subX)*one/4, -roundDiv(hy*num, den)}
	}
	var edges []edge
	add := func(a, b pt) {
		switch {
		case a.y < b.y:
			edges = append(edges, edge{a.x, a.y, b.x, b.y, 1})
		case a.y > b.y:
			edges = append(edges, edge{b.x, b.y, a.x, a.y, -1})
		}
	}
	var start, cur pt
	open := false
	for _, s := range out.Segments {
		switch s.Op {
		case ot.SegmentOpMoveTo:
			if open {
				add(cur, start)
			}
			start, cur, open = conv(s.Args[0]), conv(s.Args[0]), true
		case ot.SegmentOpLineTo:
			p := conv(s.Args[0])
			add(cur, p)
			cur = p
		case ot.SegmentOpQuadTo:
			c, p := conv(s.Args[0]), conv(s.Args[1])
			n := steps(abs(cur.x-2*c.x+p.x)+abs(cur.y-2*c.y+p.y), 8)
			prev := cur
			for i := int64(1); i <= n; i++ {
				a, b := n-i, i
				q := pt{
					roundDiv(cur.x*a*a+2*c.x*a*b+p.x*b*b, n*n),
					roundDiv(cur.y*a*a+2*c.y*a*b+p.y*b*b, n*n),
				}
				add(prev, q)
				prev = q
			}
			cur = p
		case ot.SegmentOpCubeTo:
			c1, c2, p := conv(s.Args[0]), conv(s.Args[1]), conv(s.Args[2])
			dd := max(abs(cur.x-2*c1.x+c2.x)+abs(cur.y-2*c1.y+c2.y), abs(c1.x-2*c2.x+p.x)+abs(c1.y-2*c2.y+p.y))
			n := steps(dd*3, 8) // bound for cubics: 3/4 * max second difference
			prev := cur
			for i := int64(1); i <= n; i++ {
				a, b := n-i, i
				q := pt{
					roundDiv(cur.x*a*a*a+3*c1.x*a*a*b+3*c2.x*a*b*b+p.x*b*b*b, n*n*n),
					roundDiv(cur.y*a*a*a+3*c1.y*a*a*b+3*c2.y*a*b*b+p.y*b*b*b, n*n*n),
				}
				add(prev, q)
				prev = q
			}
			cur = p
		}
	}
	if open {
		add(cur, start)
	}
	if len(edges) == 0 {
		return &paint.GlyphMask{}
	}
	minX, minY, maxX, maxY := edges[0].x0, edges[0].y0, edges[0].x0, edges[0].y1
	for _, e := range edges {
		minX, maxX = min(minX, e.x0, e.x1), max(maxX, e.x0, e.x1)
		minY, maxY = min(minY, e.y0), max(maxY, e.y1)
	}
	px0, py0 := floorDiv(minX, one), floorDiv(minY, one)
	px1, py1 := floorDiv(maxX+one-1, one), floorDiv(maxY+one-1, one)
	w, h := int(px1-px0), int(py1-py0)
	if w <= 0 || h <= 0 {
		return &paint.GlyphMask{}
	}
	acc := make([]int32, w*h) // covered length in 1/256 px, summed over sub-rows
	type cross struct {
		x   int64
		dir int
	}
	var xs []cross
	for row := 0; row < h; row++ {
		for k := 0; k < subRows; k++ {
			y := (py0+int64(row))*one + int64(k)*one/subRows + one/(2*subRows)
			xs = xs[:0]
			for _, e := range edges {
				if y < e.y0 || y >= e.y1 {
					continue
				}
				x := e.x0 + floorDiv((y-e.y0)*(e.x1-e.x0), e.y1-e.y0)
				xs = append(xs, cross{x, e.dir})
			}
			// Insertion sort: crossings per row are few.
			for i := 1; i < len(xs); i++ {
				for j := i; j > 0 && (xs[j].x < xs[j-1].x || xs[j].x == xs[j-1].x && xs[j].dir < xs[j-1].dir); j-- {
					xs[j], xs[j-1] = xs[j-1], xs[j]
				}
			}
			wind := 0
			line := acc[row*w : row*w+w]
			for i := 0; i+1 < len(xs); i++ {
				wind += xs[i].dir
				if wind == 0 {
					continue
				}
				span(line, xs[i].x-px0*one, xs[i+1].x-px0*one)
			}
		}
	}
	m := &paint.GlyphMask{X: int(px0), Y: int(py0), W: w, H: h, Pix: make([]uint8, w*h)}
	const full = one * subRows
	for i, a := range acc {
		m.Pix[i] = uint8((int64(a)*255 + full/2) / full)
	}
	return m
}

// span adds the coverage of [a, b) (1/256 px, relative to the mask's left
// edge) to one row of the accumulator.
func span(line []int32, a, b int64) {
	lim := int64(len(line)) * one
	a, b = max(a, 0), min(b, lim)
	if a >= b {
		return
	}
	ia, ib := a>>fracBits, (b-1)>>fracBits
	if ia == ib {
		line[ia] += int32(b - a)
		return
	}
	line[ia] += int32((ia+1)*one - a)
	for i := ia + 1; i < ib; i++ {
		line[i] += one
	}
	line[ib] += int32(b - ib*one)
}

// steps returns the number of line segments for a curve whose second
// difference (in 1/256 px) is dd, so that the error dd/(c*n^2) stays
// under 1/16 px.
func steps(dd int64, c int64) int64 {
	tol := int64(one / 16)
	n := isqrt(dd/(c*tol)) + 1
	return min(max(n, 1), 64)
}

func isqrt(v int64) int64 {
	if v <= 0 {
		return 0
	}
	x := int64(1) << ((bitsLen(uint64(v)) + 1) / 2)
	for {
		y := (x + v/x) / 2
		if y >= x {
			return x
		}
		x = y
	}
}

func bitsLen(v uint64) int {
	n := 0
	for v != 0 {
		v >>= 1
		n++
	}
	return n
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// roundDiv returns a/b rounded half away from zero (b > 0).
func roundDiv(a, b int64) int64 {
	if a < 0 {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}
