package flexui

import (
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
