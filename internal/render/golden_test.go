package render

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// RenderHTML golden tests. Each document in testdata/render/docs is a
// standalone HTML file whose first line is `<!-- size: WxH -->`. It is
// rendered at scale 1 (and the docs listed in hidpiDocs also at scale 2)
// and compared with testdata/render/golden/<name>[@2x].png. Regenerate with
//
//	go test ./internal/render -run TestGolden -update
//
// and look at the PNGs before committing.

var update = flag.Bool("update", false, "rewrite golden PNGs")

const goldenTolerance = 2

var (
	docsDir   = filepath.Join("..", "..", "testdata", "render", "docs")
	goldenDir = filepath.Join("..", "..", "testdata", "render", "golden")
	hidpiDocs = map[string]bool{"app_shell": true, "radius": true, "text_card": true, "zindex_clip": true}
	sizeRE    = regexp.MustCompile(`^<!-- size: (\d+)x(\d+) -->`)
)

type doc struct {
	name string
	w, h int
	html string
}

func loadDocs(t testing.TB) []doc {
	ents, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []doc
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(docsDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		m := sizeRE.FindSubmatch(b)
		if m == nil {
			t.Fatalf("%s: missing size comment", e.Name())
		}
		w, _ := strconv.Atoi(string(m[1]))
		h, _ := strconv.Atoi(string(m[2]))
		out = append(out, doc{strings.TrimSuffix(e.Name(), ".html"), w, h, string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

type variant struct {
	file  string
	scale float64
}

func variants(d doc) []variant {
	v := []variant{{d.name, 1}}
	if hidpiDocs[d.name] {
		v = append(v, variant{d.name + "@2x", 2})
	}
	return v
}

func TestGolden(t *testing.T) {
	docs := loadDocs(t)
	if len(docs) < 10 {
		t.Fatalf("only %d documents", len(docs))
	}
	for _, d := range docs {
		for _, v := range variants(d) {
			t.Run(v.file, func(t *testing.T) {
				got, warns := Render(d.html, "", d.w, d.h, v.scale)
				if len(warns) != 0 {
					t.Errorf("warnings: %q", warns)
				}
				path := filepath.Join(goldenDir, v.file+".png")
				if *update {
					os.MkdirAll(goldenDir, 0o755)
					if err := writePNG(path, got); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := readPNG(path)
				if err != nil {
					t.Fatalf("%v (run with -update)", err)
				}
				st := compare(got, want)
				if st.over(goldenTolerance) > 0 {
					fail := filepath.Join(goldenDir, "_failed")
					os.MkdirAll(fail, 0o755)
					writePNG(filepath.Join(fail, v.file+".got.png"), got)
					t.Errorf("%d pixels differ by more than %d (max %d); see %s", st.over(goldenTolerance), goldenTolerance, st.max, fail)
				}
			})
		}
	}
}

// diffStats summarizes per-pixel differences (max over R, G, B, A).
type diffStats struct {
	pixels  int
	max     int
	sum     int64 // sum of per-channel absolute differences (R, G, B)
	hist    [256]int
	sizeBad bool
}

func (s diffStats) mean() float64 { return float64(s.sum) / float64(3*max(s.pixels, 1)) }

// over returns the number of pixels whose largest channel difference
// exceeds tol.
func (s diffStats) over(tol int) int {
	if s.sizeBad {
		return s.pixels
	}
	n := 0
	for d := tol + 1; d < 256; d++ {
		n += s.hist[d]
	}
	return n
}

func compare(a, b *image.RGBA) diffStats {
	var s diffStats
	s.pixels = len(a.Pix) / 4
	if a.Rect.Size() != b.Rect.Size() {
		s.sizeBad, s.max = true, 255
		return s
	}
	for i := 0; i < len(a.Pix); i += 4 {
		m := 0
		for k := 0; k < 4; k++ {
			d := int(a.Pix[i+k]) - int(b.Pix[i+k])
			if d < 0 {
				d = -d
			}
			if k < 3 {
				s.sum += int64(d)
			}
			m = max(m, d)
		}
		s.hist[m]++
		s.max = max(s.max, m)
	}
	return s
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
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			rgba.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return rgba, nil
}
