package jpeg

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/draw"
	stdjpeg "image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The test images are Go 1.24's src/image/testdata JPEGs (BSD licence,
// LICENSE): every chroma subsampling, baseline and progressive, restart
// markers, gray, RGB and Adobe CMYK.

var update = flag.Bool("update", false, "rewrite testdata/hashes.txt (only together with a deliberate decoder change)")

func decodeAll(t *testing.T, dec func([]byte) (*image.RGBA, error)) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "*.jpeg"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no test images: %v", err)
	}
	var lines []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m, err := dec(data)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		lines = append(lines, fmt.Sprintf("%s %dx%d %x", filepath.Base(f), m.Rect.Dx(), m.Rect.Dy(), sha256.Sum256(m.Pix)))
	}
	sort.Strings(lines)
	return lines
}

func ours(data []byte) (*image.RGBA, error) { return DecodeRGBA(bytes.NewReader(data)) }

// TestHashes pins the decoded pixels: they must never change with the Go
// version or the platform (spec/SUBSET.md §5).
func TestHashes(t *testing.T) {
	got := strings.Join(decodeAll(t, ours), "\n") + "\n"
	path := filepath.Join("testdata", "hashes.txt")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s/%s %s: decoded pixels changed\ngot:\n%swant:\n%s", runtime.GOOS, runtime.GOARCH, runtime.Version(), got, want)
	}
}

// TestMatchesGo124 checks the copy against the original: with Go 1.24 the
// standard library decoder plus draw.Draw must give identical bytes. On
// other Go versions the standard library decodes differently, which is
// why this package exists, so the comparison is skipped there.
func TestMatchesGo124(t *testing.T) {
	if !strings.HasPrefix(runtime.Version(), "go1.24") {
		t.Skipf("reference is Go 1.24's image/jpeg; running %s", runtime.Version())
	}
	std := func(data []byte) (*image.RGBA, error) {
		m, err := stdjpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		b := m.Bounds()
		out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(out, out.Rect, m, b.Min, draw.Src)
		return out, nil
	}
	got, want := decodeAll(t, ours), decodeAll(t, std)
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("differs from Go 1.24:\n got  %s\n want %s", got[i], want[i])
		}
	}
}
