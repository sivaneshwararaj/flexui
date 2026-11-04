package paint

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Golden-image tests. Each scene is recorded into a List, replayed into a
// Raster, and compared with testdata/paint/<name>.png. Regenerate with
//
//	go test ./internal/paint -run TestGolden -update
//
// and look at the PNGs before committing them. On mismatch the actual
// image and a diff are written to testdata/paint/_failed/ (gitignored).

var update = flag.Bool("update", false, "rewrite golden PNGs")

// goldenTolerance is the largest per-channel difference accepted. Output
// is deterministic, so this only absorbs deliberate tiny changes.
const goldenTolerance = 2

type scene struct {
	name  string
	w, h  int // logical size
	scale float64
	draw  func(c Canvas)
}

func px(v float64) geom.Unit { return geom.Unit(v*64 + 0.5) }

func rc(x, y, w, h float64) geom.Rect { return geom.Rect{X: px(x), Y: px(y), W: px(w), H: px(h)} }

func edges(t, r, b, l float64) geom.Edges {
	return geom.Edges{Top: px(t), Right: px(r), Bottom: px(b), Left: px(l)}
}

var (
	white  = Opaque(255, 255, 255)
	ink    = Opaque(30, 34, 42)
	blue   = Opaque(52, 120, 246)
	red    = Opaque(230, 70, 60)
	green  = Opaque(40, 170, 100)
	yellow = Opaque(250, 200, 40)
	gray   = Opaque(200, 204, 210)
)

func bg(c Canvas, w, h float64) { c.FillRRect(Rect(rc(0, 0, w, h)), Opaque(246, 247, 249)) }

func card(c Canvas, x, y float64) {
	r := Uniform(rc(x, y, 120, 70), px(10))
	c.BoxShadow(r, Shadow{DY: px(4), Blur: px(14), Color: Color{0, 0, 0, 70}})
	c.FillRRect(r, white)
	c.PushClip(r)
	c.FillRRect(Rect(rc(x, y, 120, 22)), blue)
	c.PopClip()
	c.FillRRect(Uniform(rc(x+10, y+34, 70, 8), px(4)), gray)
	c.FillRRect(Uniform(rc(x+10, y+48, 46, 8), px(4)), gray)
	c.Border(r, edges(1, 1, 1, 1), Color{0, 0, 0, 40})
}

var scenes = []scene{
	{"rects", 240, 140, 1, func(c Canvas) {
		bg(c, 240, 140)
		c.FillRRect(Rect(rc(10, 10, 60, 40)), blue)
		c.FillRRect(Rect(rc(40, 30, 60, 40)), Color{230, 70, 60, 128})
		c.FillRRect(Rect(rc(70, 50, 60, 40)), Color{40, 170, 100, 160})
		// Snapping: x = 10.25, 10.5, 10.75 and fractional widths.
		for i, f := range []float64{0, 0.25, 0.5, 0.75} {
			c.FillRRect(Rect(rc(150+f, 10+float64(i)*14, 30.4, 10)), ink)
		}
		// Adjacent fractional boxes share edges: no seams, no overlap.
		x := 150.0
		for i := 0; i < 6; i++ {
			col := blue
			if i%2 == 1 {
				col = yellow
			}
			c.FillRRect(Rect(rc(x, 70, 10.3, 20)), col)
			x += 10.3
		}
		// Hairlines and a translucent overlay.
		c.FillRRect(Rect(rc(10, 110, 220, 1)), ink)
		c.FillRRect(Rect(rc(10, 114, 220, 0.5)), ink)
		c.FillRRect(Rect(rc(10, 100, 220, 30)), Color{52, 120, 246, 40})
	}},
	{"rounded", 260, 150, 1, func(c Canvas) {
		bg(c, 260, 150)
		c.FillRRect(Uniform(rc(10, 10, 70, 50), px(8)), blue)
		c.FillRRect(Uniform(rc(95, 10, 50, 50), px(25)), red)     // circle
		c.FillRRect(Uniform(rc(160, 20, 90, 30), px(100)), green) // pill (radii normalized)
		c.FillRRect(Uniform(rc(10, 75, 40, 40), px(1.5)), ink)    // tiny radius
		rr := RRect{Rect: rc(65, 75, 80, 60)}                     // elliptical + mixed corners
		rr.Radii = [4]Radius{{px(30), px(15)}, {0, 0}, {px(20), px(20)}, {px(5), px(25)}}
		c.FillRRect(rr, yellow)
		c.FillRRect(Uniform(rc(160, 75, 90, 60), px(16)), Color{52, 120, 246, 120})
		c.FillRRect(Uniform(rc(180, 90, 50, 30), px(15)), Color{230, 70, 60, 160})
	}},
	{"borders", 260, 150, 1, func(c Canvas) {
		bg(c, 260, 150)
		c.FillRRect(Rect(rc(10, 10, 70, 50)), white)
		c.Border(Rect(rc(10, 10, 70, 50)), edges(2, 2, 2, 2), ink)
		c.Border(Rect(rc(95, 10, 70, 50)), edges(1, 4, 8, 12), blue)
		c.Border(Rect(rc(180, 10, 70, 50)), edges(0.4, 0.4, 0.4, 0.4), ink) // thin -> 1px
		rr := Uniform(rc(10, 75, 70, 60), px(14))
		c.FillRRect(rr, white)
		c.Border(rr, edges(3, 3, 3, 3), red)
		c.Border(Uniform(rc(95, 75, 70, 60), px(18)), edges(2, 10, 2, 10), green) // elliptical inner
		c.FillRRect(Uniform(rc(180, 75, 70, 60), px(30)), yellow)
		c.Border(Uniform(rc(180, 75, 70, 60), px(30)), edges(6, 6, 6, 6), Color{30, 34, 42, 128})
	}},
	{"shadows", 300, 170, 1, func(c Canvas) {
		bg(c, 300, 170)
		r1 := Uniform(rc(20, 20, 70, 50), px(8))
		c.BoxShadow(r1, Shadow{DY: px(6), Blur: px(16), Color: Color{0, 0, 0, 90}})
		c.FillRRect(r1, white)
		r2 := Rect(rc(115, 20, 70, 50))
		c.BoxShadow(r2, Shadow{DX: px(6), DY: px(6), Color: Color{52, 120, 246, 255}}) // hard
		c.FillRRect(r2, white)
		r3 := Uniform(rc(210, 20, 70, 50), px(25))
		c.BoxShadow(r3, Shadow{Spread: px(4), Blur: px(4), Color: Color{230, 70, 60, 160}})
		c.FillRRect(r3, white)
		// Transparent box: the shadow is not drawn under it.
		r4 := Uniform(rc(20, 100, 70, 50), px(6))
		c.BoxShadow(r4, Shadow{Blur: px(10), Spread: px(2), Color: Color{0, 0, 0, 140}})
		c.Border(r4, edges(1, 1, 1, 1), gray)
		r5 := Uniform(rc(130, 100, 140, 50), px(12))
		c.BoxShadow(r5, Shadow{DY: px(10), Blur: px(30), Spread: px(-6), Color: Color{0, 0, 0, 160}})
		c.FillRRect(r5, white)
	}},
	{"opacity", 260, 130, 1, func(c Canvas) {
		bg(c, 260, 130)
		// Group opacity: the overlap is not visible.
		c.PushOpacity(128)
		c.FillRRect(Rect(rc(10, 10, 60, 60)), blue)
		c.FillRRect(Rect(rc(40, 40, 60, 60)), red)
		c.PopOpacity()
		// Per-element opacity: the overlap shows.
		c.FillRRect(Rect(rc(140, 10, 60, 60)), Color{52, 120, 246, 128})
		c.FillRRect(Rect(rc(170, 40, 60, 60)), Color{230, 70, 60, 128})
		// Nested groups: 50% of 50%.
		c.PushOpacity(128)
		c.PushOpacity(128)
		c.FillRRect(Uniform(rc(105, 100, 50, 20), px(10)), ink)
		c.PopOpacity()
		c.PopOpacity()
	}},
	{"clip", 260, 150, 1, func(c Canvas) {
		bg(c, 260, 150)
		// Rect clip.
		c.PushClip(Rect(rc(10, 10, 70, 60)))
		c.FillRRect(Uniform(rc(-10, 30, 120, 60), px(30)), blue)
		c.PopClip()
		c.Border(Rect(rc(10, 10, 70, 60)), edges(1, 1, 1, 1), ink)
		// Rounded clip with overflowing content (like an image in a card).
		r := Uniform(rc(100, 10, 70, 60), px(18))
		c.PushClip(r)
		c.FillRRect(Rect(rc(90, 0, 90, 30)), red)
		c.FillRRect(Rect(rc(90, 30, 90, 50)), yellow)
		c.PopClip()
		// Nested: rect inside a circle; shadows and borders are clipped.
		c.PushClip(Uniform(rc(190, 10, 60, 60), px(30)))
		c.PushClip(Rect(rc(200, 0, 60, 50)))
		c.FillRRect(Rect(rc(180, 0, 90, 90)), green)
		c.PopClip()
		c.PopClip()
		// Clipped group opacity.
		c.PushClip(Uniform(rc(10, 85, 240, 55), px(12)))
		c.PushOpacity(160)
		card(c, 30, 80)
		c.PopOpacity()
		c.PopClip()
	}},
	{"card_2x", 160, 100, 2, func(c Canvas) {
		bg(c, 160, 100)
		card(c, 20, 15)
	}},
	{"card_1_5x", 160, 100, 1.5, func(c Canvas) {
		bg(c, 160, 100)
		card(c, 20.3, 15.7)
	}},
}

func renderScene(s scene) *image.RGBA {
	var l List
	s.draw(&l)
	img := image.NewRGBA(image.Rect(0, 0, int(float64(s.w)*s.scale+0.5), int(float64(s.h)*s.scale+0.5)))
	var r Raster
	r.Begin(img, Scale64(s.scale))
	l.Replay(&r)
	r.End()
	return img
}

func TestGolden(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "paint")
	for _, s := range scenes {
		t.Run(s.name, func(t *testing.T) {
			got := renderScene(s)
			path := filepath.Join(dir, s.name+".png")
			if *update {
				if err := writePNG(path, got); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := readPNG(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create)", err)
			}
			if n, maxd := diff(got, want); n > 0 {
				fail := filepath.Join(dir, "_failed")
				os.MkdirAll(fail, 0o755)
				writePNG(filepath.Join(fail, s.name+".got.png"), got)
				writePNG(filepath.Join(fail, s.name+".diff.png"), diffImage(got, want))
				t.Errorf("%d pixels differ (max channel diff %d); see %s", n, maxd, fail)
			}
		})
	}
}

// TestDeterministic renders every scene twice with a reused Raster and
// checks identical bytes (buffers and pools must not leak state).
func TestDeterministic(t *testing.T) {
	var r Raster
	for _, s := range scenes {
		a := renderScene(s)
		var l List
		s.draw(&l)
		b := image.NewRGBA(a.Rect)
		r.Begin(b, Scale64(s.scale))
		l.Replay(&r)
		r.End()
		if !bytes.Equal(a.Pix, b.Pix) {
			t.Errorf("%s: reused raster differs", s.name)
		}
	}
}

func writePNG(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func readPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba, nil
}

func diff(a, b *image.RGBA) (n, maxd int) {
	if a.Rect != b.Rect {
		return len(a.Pix) / 4, 255
	}
	for i := 0; i < len(a.Pix); i += 4 {
		bad := false
		for k := 0; k < 4; k++ {
			d := int(a.Pix[i+k]) - int(b.Pix[i+k])
			if d < 0 {
				d = -d
			}
			maxd = max(maxd, d)
			if d > goldenTolerance {
				bad = true
			}
		}
		if bad {
			n++
		}
	}
	return n, maxd
}

func diffImage(a, b *image.RGBA) *image.RGBA {
	out := image.NewRGBA(a.Rect)
	for i := 0; i+3 < len(a.Pix) && i+3 < len(b.Pix); i += 4 {
		d := 0
		for k := 0; k < 4; k++ {
			d = max(d, abs(int(a.Pix[i+k])-int(b.Pix[i+k])))
		}
		v := uint8(min(255, d*8))
		out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = v, 0, 0, 255
	}
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
