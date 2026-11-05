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
