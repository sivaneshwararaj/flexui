package render

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/text"
)

// TestLazyFontPacks checks that font packs are parsed only when text
// falls back to them (spec/SUBSET.md §7.4). It runs in a child process
// because the other tests of this package load every pack.
func TestLazyFontPacks(t *testing.T) {
	if os.Getenv("FLEXUI_LAZY_CHILD") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestLazyFontPacks$", "-test.count=1")
		cmd.Env = append(os.Environ(), "FLEXUI_LAZY_CHILD=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child: %v\n%s", err, out)
		}
		return
	}
	steps := []struct{ html, want string }{
		{`<p>Hello, world</p>`, ""},
		{`<p>வணக்கம்</p>`, "NotoSansTamil-Regular"},
		{`<p style="font-weight: 700">வணக்கம்</p>`, "NotoSansTamil-Regular NotoSansTamil-Bold"},
		// Hebrew falls through Tamil (loaded), Devanagari and Arabic first.
		{`<p>שלום</p>`, "NotoSansTamil-Regular NotoSansTamil-Bold NotoSansDevanagari-Regular NotoSansArabic-Regular NotoSansHebrew-Regular"},
		// The CJK pack is last: only text no other font covers reaches it.
		{`<p>你好</p>`, "NotoSansTamil-Regular NotoSansTamil-Bold NotoSansDevanagari-Regular NotoSansArabic-Regular NotoSansHebrew-Regular NotoSansSC-Regular"},
	}
	for _, s := range steps {
		if _, w := Render(s.html, "", 200, 60, 1); len(w) != 0 {
			t.Errorf("%s: warnings %q", s.html, w)
		}
		if got := strings.Join(text.LoadedPackFaces(), " "); got != s.want {
			t.Errorf("after %s: loaded %q, want %q", s.html, got, s.want)
		}
	}
}
