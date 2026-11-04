package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChromeDiff compares RenderHTML output with Chromium screenshots of
// the same documents (testdata/render/chrome, made by `make crosscheck`
// with scripts/screenshot.cjs). For every document it reports the
// maximum and mean per-channel error and the share of pixels that differ
// by more than 8 and 32 levels. The default bounds are near-exact: mean
// error at most chromeMaxMean and no pixel off by more than 32. Every
// document that exceeds them is listed in acceptedChromeDiffs with the
// deviation from spec/SUBSET.md that explains it, and with bounds about
// 1.5x its measured error, so it cannot silently get worse.
//
// The images are written to testdata/render/chrome-actual (gitignored) for
// inspection.

const (
	chromeMaxMean = 0.05 // mean absolute channel error, in levels of 255
	chromeMaxBad  = 0.0  // share of pixels with an error > 32
)

// acceptedChromeDiffs maps a document variant to the deviations that
// explain its difference, and the bounds it must stay within. Measured
// values at the time of writing are in the comments (mean / share > 32).
var acceptedChromeDiffs = map[string]accepted{
	"app_shell":       {"D17 shadow blur, D18 curve AA", 0.12, 0.0005},                  // 0.078 / 0.000%
	"app_shell@2x":    {"D17 shadow blur, D18 curve AA", 0.16, 0.0003},                  // 0.103 / 0.016%
	"fractional":      {"D18 curve AA", 0.10, 0.0009},                                   // 0.066 / 0.056%
	"grid_wrap":       {"D17 shadow blur, D18 curve AA", 0.28, 0.0010},                  // 0.183 / 0.067%
	"opacity":         {"D19 group-opacity rounding", 0.23, 0},                          // 0.149 / 0.000%, max 2
	"overflow":        {"D18 curve AA (rounded clips)", 0.18, 0.0023},                   // 0.120 / 0.148%
	"radius":          {"D18 curve AA", 0.26, 0.0020},                                   // 0.170 / 0.128%
	"radius@2x":       {"D18 curve AA", 0.09, 0.0007},                                   // 0.059 / 0.041%
	"shadows":         {"D17 shadow blur, D18 curve AA", 0.71, 0.0009},                  // 0.473 / 0.055%
	"specificity":     {"D18 curve AA", 0.08, 0.0001},                                   // 0.053 / 0.003%
	"text_card":       {"D21 glyph AA, D18 curve AA", 0.47, 0.0003},                     // 0.313 / 0.017%
	"text_card@2x":    {"D22 device-pixel font metrics, D21", 0.90, 0.0105},             // 0.599 / 0.696%
	"text_ellipsis":   {"D21 glyph AA, D18 curve AA", 0.59, 0.0016},                     // 0.394 / 0.105%
	"text_paragraphs": {"D21 glyph AA", 0.76, 0.0012},                                   // 0.505 / 0.081%
	"controls":        {"D18 curve AA, D21 glyph AA", 0.24, 0.0007},                     // 0.160 / 0.044%
	"images":          {"D28 img radius clip, D29 image decoding/filter", 0.50, 0.0030}, // 0.334 / 0.199%
	"inline_text":     {"D21 glyph AA", 0.80, 0.0022},                                   // 0.535 / 0.145%
	"zindex_clip@2x":  {"D18 curve AA (rounded clip)", 0.05, 0.0001},                    // 0.013 / 0.004%
}

type accepted struct {
	deviation string
	maxMean   float64
	maxBad    float64
}

func TestChromeDiff(t *testing.T) {
	chromeDir := filepath.Join("..", "..", "testdata", "render", "chrome")
	actualDir := filepath.Join("..", "..", "testdata", "render", "chrome-actual")
	os.MkdirAll(actualDir, 0o755)
	var report strings.Builder
	fmt.Fprintf(&report, "\n%-18s %5s %7s %8s %8s  %s\n", "document", "max", "mean", ">8", ">32", "verdict")
	for _, d := range loadDocs(t) {
		for _, v := range variants(d) {
			want, err := readPNG(filepath.Join(chromeDir, v.file+".png"))
			if err != nil {
				t.Errorf("%s: no Chromium screenshot (%v); run `make crosscheck`", v.file, err)
				continue
			}
			got, _ := Render(d.html, "", d.w, d.h, v.scale)
			writePNG(filepath.Join(actualDir, v.file+".png"), got)
			st := compare(got, want)
			bad8 := float64(st.over(8)) / float64(st.pixels)
			bad32 := float64(st.over(32)) / float64(st.pixels)
			verdict := "ok"
			maxMean, maxBad := chromeMaxMean, chromeMaxBad
			if acc, ok := acceptedChromeDiffs[v.file]; ok {
				maxMean, maxBad = acc.maxMean, acc.maxBad
				verdict = "accepted (" + acc.deviation + ")"
			}
			if st.mean() > maxMean || bad32 > maxBad {
				verdict = "FAIL"
				t.Errorf("%s: mean %.3f (limit %.3f), >32: %.3f%% (limit %.3f%%)",
					v.file, st.mean(), maxMean, 100*bad32, 100*maxBad)
			}
			fmt.Fprintf(&report, "%-18s %5d %7.3f %7.3f%% %7.3f%%  %s\n",
				v.file, st.max, st.mean(), 100*bad8, 100*bad32, verdict)
		}
	}
	t.Log(report.String())
}
