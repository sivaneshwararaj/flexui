package paint

import (
	"image"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

func TestSnapRule(t *testing.T) {
	for _, c := range []struct {
		units int64
		want  int
	}{
		{0, 0}, {31, 0}, {32, 1}, {64, 1}, {95, 1}, {96, 2}, {-32, 0}, {-33, -1},
	} {
		if got := snap(c.units); got != c.want {
			t.Errorf("snap(%d) = %d, want %d", c.units, got, c.want)
		}
	}
}

// Boxes that touch in layout must touch exactly in pixels: every pixel of
// a row tiled with fractional-width boxes is painted exactly once.
func TestAdjacentBoxesTile(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		img := image.NewRGBA(image.Rect(0, 0, 400, 4))
		var r Raster
		r.Begin(img, Scale64(scale))
		x := px(0.3)
		for i := 0; i < 37; i++ {
			w := px(7.17)
			r.FillRRect(Rect(geom.Rect{X: x, Y: 0, W: w, H: px(2)}), Color{0, 0, 0, 100})
			x += w
		}
		end := min(snap(r.dev(x)), 400)
		for X := snap(r.dev(px(0.3))); X < end; X++ {
			if a := img.Pix[img.PixOffset(X, 0)+3]; a != 100 {
				t.Fatalf("scale %v: pixel %d alpha %d (gap or overlap)", scale, X, a)
			}
		}
	}
}

func TestBorderWidthsUniform(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	var r Raster
	r.Begin(img, 64)
	// A fractional box with a 1.4px border: every side must be exactly 1px.
	r.Border(Rect(geom.Rect{X: px(5.3), Y: px(5.6), W: px(20.5), H: px(20.2)}), edges(1.4, 1.4, 1.4, 1.4), Opaque(0, 0, 0))
	count := func(x0, y0, dx, dy int) int {
		n := 0
		for x, y := x0, y0; x >= 0 && y >= 0 && x < 40 && y < 40; x, y = x+dx, y+dy {
			if img.Pix[img.PixOffset(x, y)+3] != 0 {
				n++
			}
		}
		return n
	}
	// Scan inward from the middle of each side.
	if a, b, c, d := count(15, 0, 0, 1), count(15, 39, 0, -1), count(0, 15, 1, 0), count(39, 15, -1, 0); a != 2 || b != 2 || c != 2 || d != 2 {
		// Each scan crosses the border twice (outer side and far side).
		t.Errorf("border thickness per scan: %d %d %d %d, want 2 each", a, b, c, d)
	}
}

func TestInEllipse(t *testing.T) {
	r := rad{64 * 1000, 64 * 500}
	if !inEllipse(0, 0, r) || !inEllipse(r.x, 0, r) || !inEllipse(0, r.y, r) {
		t.Error("boundary points should be inside")
	}
	if inEllipse(r.x, 1, r) || inEllipse(r.x/2+r.x/3, r.y/2+r.y/3, r) {
		t.Error("outside points reported inside")
	}
	// Large radii must not overflow.
	big := rad{1 << 22, 1 << 22}
	if !inEllipse(big.x/2, big.y/2, big) {
		t.Error("large radius")
	}
}

func TestIsqrt(t *testing.T) {
	for _, v := range []uint64{0, 1, 2, 3, 4, 15, 16, 17, 1 << 40, 1<<62 + 12345} {
		s := isqrt(v)
		if s*s > v || (s+1)*(s+1) <= v {
			t.Errorf("isqrt(%d) = %d", v, s)
		}
	}
}

func TestListReplayMatchesDirect(t *testing.T) {
	for _, s := range scenes {
		direct := image.NewRGBA(renderScene(s).Rect)
		var r Raster
		r.Begin(direct, Scale64(s.scale))
		s.draw(&r)
		r.End()
		if got := renderScene(s); string(got.Pix) != string(direct.Pix) {
			t.Errorf("%s: replayed list differs from direct drawing", s.name)
		}
	}
}

// Steady-state frames must not allocate: buffers, layers and the display
// list are reused.
func TestNoAllocsPerFrame(t *testing.T) {
	var l List
	var r Raster
	s := scenes[len(scenes)-3] // clip scene: clips, layers, shadows, borders
	img := image.NewRGBA(image.Rect(0, 0, s.w, s.h))
	frame := func() {
		l.Reset()
		s.draw(&l)
		r.Begin(img, 64)
		l.Replay(&r)
		r.End()
	}
	frame()
	if n := testing.AllocsPerRun(5, frame); n != 0 {
		t.Errorf("%v allocations per frame", n)
	}
}

func BenchmarkCardFrame(b *testing.B) {
	s := scenes[len(scenes)-2]
	var l List
	s.draw(&l)
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	var r Raster
	for i := 0; i < b.N; i++ {
		r.Begin(img, Scale64(2))
		l.Replay(&r)
		r.End()
	}
}
