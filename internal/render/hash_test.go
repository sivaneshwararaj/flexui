package render

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestHashes is the determinism gate (spec/SUBSET.md §5): every golden
// document renders to exactly the bytes recorded in
// testdata/render/hashes.txt, on every GOOS and GOARCH. It is run on Linux
// for amd64 and 386 by scripts/check-portability.sh; on a Mac, run
//
//	go test ./internal/render -run TestHashes
//
// Regenerate with -update only together with the goldens.
func TestHashes(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "render", "hashes.txt")
	var lines []string
	for _, d := range loadDocs(t) {
		for _, v := range variants(d) {
			img, _ := Render(d.html, "", d.w, d.h, v.scale)
			lines = append(lines, fmt.Sprintf("%s %x", v.file, sha256.Sum256(img.Pix)))
		}
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if got != string(want) {
		wl := strings.Split(string(want), "\n")
		for i, l := range strings.Split(got, "\n") {
			if i >= len(wl) || l != wl[i] {
				t.Errorf("%s/%s: %q, want %q", runtime.GOOS, runtime.GOARCH, l, wl[min(i, len(wl)-1)])
			}
		}
	}
}
