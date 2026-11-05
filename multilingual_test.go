package flexui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/testtext"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// TestPasteMultilingual pastes complex-script, right-to-left, combining
// and astral strings through the clipboard interface into an input and
// checks that the value and a copy back out are byte-identical.
func TestPasteMultilingual(t *testing.T) {
	for _, smp := range testtext.Samples {
		t.Run(smp.Name, func(t *testing.T) {
			f := newFormApp()
			s := headless.Start(f.app)
			s.Click(center(t, f.app, "name"))
			if err := s.Clipboard.WriteText(smp.Text); err != nil {
				t.Fatal(err)
			}
			s.Key("v", platform.ModCtrl)
			if got := f.app.Value("name"); got != smp.Text {
				t.Fatalf("value after paste:\n got  %+q\n want %+q", got, smp.Text)
			}
			if f.Name != smp.Text {
				t.Errorf("oninput value %+q", f.Name)
			}
			s.Clipboard.WriteText("")
			s.Key("a", platform.ModCtrl)
			s.Key("c", platform.ModCtrl)
			if got, _ := s.Clipboard.ReadText(); got != smp.Text {
				t.Errorf("copied back:\n got  %+q\n want %+q", got, smp.Text)
			}
			// A cut and paste returns the same bytes too.
			s.Key("x", platform.ModCtrl)
			s.Key("v", platform.ModCtrl)
			if got := f.app.Value("name"); got != smp.Text {
				t.Errorf("after cut and paste: %+q", got)
			}
			s.Frame() // renders without panicking
		})
	}
}

// TestMultilingualPages renders testdata/multilingual/*.html (all font
// packs registered, see fonts_test.go) and compares the PNGs.
func TestMultilingualPages(t *testing.T) {
	docs, _ := filepath.Glob(filepath.Join("testdata", "multilingual", "*.html"))
	if len(docs) == 0 {
		t.Fatal("no pages")
	}
	for _, p := range docs {
		name := strings.TrimSuffix(filepath.Base(p), ".html")
		t.Run(name, func(t *testing.T) {
			src, _ := os.ReadFile(p)
			var w, h int
			if _, err := fmt.Sscanf(string(src), "<!-- size: %dx%d -->", &w, &h); err != nil {
				t.Fatal(err)
			}
			if warns := CheckHTML(string(src), ""); len(warns) != 0 {
				t.Errorf("warnings: %q", warns)
			}
			checkGoldenAt(t, filepath.Join("testdata", "multilingual", name+".png"), RenderHTML(string(src), "", w, h))
		})
	}
}

// TestMultilingualInput types, pastes and selects multi-script text in
// an input (testdata/multilingual/input_*.png).
func TestMultilingualInput(t *testing.T) {
	f := newFormApp()
	s := headless.Start(f.app)
	s.Click(center(t, f.app, "name"))
	s.Clipboard.WriteText("தமிழ் नमस्ते")
	s.Key("v", platform.ModCtrl)
	s.Type(" ok")
	checkGoldenAt(t, filepath.Join("testdata", "multilingual", "input_typed.png"), s.Frame())
	s.Key("ArrowLeft", platform.ModCtrl)
	s.Key("ArrowLeft", platform.ModCtrl|platform.ModShift) // select "नमस्ते "
	c, a := f.app.view.InputSelection("name")
	if sel := string([]rune(f.app.Value("name"))[min(c, a):max(c, a)]); sel != "नमस्ते " {
		t.Errorf("selection %d..%d %q", c, a, sel)
	}
	checkGoldenAt(t, filepath.Join("testdata", "multilingual", "input_selected.png"), s.Frame())
}

// TestRTLInput edits a right-to-left input: right-aligned text, arrow
// keys reversed (logical movement, as Chromium), Backspace by code point
// in a cluster, a selection across directions, the caret placed by a
// click (testdata/multilingual/input_rtl_*.png).
func TestRTLInput(t *testing.T) {
	var val string
	app := New(Options{Width: 300, Height: 90})
	app.SetView(`<div class="f"><input id="he" dir="rtl" oninput="In"><input id="mix"></div>`,
		`body { padding: 12px; background-color: #f4f5f7; } .f { gap: 8px; } input { width: 260px; }`)
	app.Handle("In", func(c *Ctx) { val = c.Value })
	s := headless.Start(app)
	if w := app.Warnings(); len(w) != 0 {
		t.Fatalf("warnings %q", w)
	}
	s.Click(center(t, app, "he"))
	s.Type("שָׁלוֹם abc")
	checkGoldenAt(t, filepath.Join("testdata", "multilingual", "input_rtl_typed.png"), s.Frame())
	// ArrowRight moves backward in a right-to-left input.
	s.Key("ArrowRight", 0)
	s.Key("ArrowRight", 0)
	if c, _ := app.view.InputSelection("he"); c != 9 { // 11 code points
		t.Errorf("after two ArrowRight: caret %d, want 9", c)
	}
	s.Key("ArrowLeft", 0) // forward again
	if c, _ := app.view.InputSelection("he"); c != 10 {
		t.Errorf("after ArrowLeft: caret %d, want 10", c)
	}
	// Backspace in "שָׁ" removes one mark at a time.
	s.Key("Home", 0)
	s.Key("ArrowLeft", 0) // forward past שָׁ (one cluster)
	s.Key("Backspace", 0)
	if val != "שָלוֹם abc" {
		t.Errorf("backspace in a cluster: %+q", val)
	}
	// Select across the direction change and see it drawn in two parts.
	s.Key("End", 0)
	s.Key("ArrowRight", platform.ModShift|platform.ModCtrl)
	s.Key("ArrowRight", platform.ModShift|platform.ModCtrl)
	checkGoldenAt(t, filepath.Join("testdata", "multilingual", "input_rtl_selected.png"), s.Frame())
	// A click at the right end of the right-to-left input is the start.
	x, y, w, h, _ := app.Bounds("he")
	s.Click(x+w-6, y+h/2)
	if c, a := app.view.InputSelection("he"); c != 0 || a != 0 {
		t.Errorf("click at the right end: caret %d anchor %d, want 0", c, a)
	}
	// Mixed text in a left-to-right input keeps logical arrow movement.
	s.Click(center(t, app, "mix"))
	s.Type("ab שלום cd")
	s.Key("Home", 0)
	for k := 0; k < 4; k++ {
		s.Key("ArrowRight", 0)
	}
	if c, _ := app.view.InputSelection("mix"); c != 4 {
		t.Errorf("mixed ltr input: caret %d after 4 ArrowRight, want 4", c)
	}
}
