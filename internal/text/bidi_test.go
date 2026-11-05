package text

import (
	"fmt"
	"testing"

	"github.com/go-text/typesetting/bidi"
)

// TestRunLevels guards runLevels (reads go-text's unexported per-character
// levels) with UAX #9 results.
func TestRunLevels(t *testing.T) {
	var p bidi.Paragraph
	for _, c := range []struct {
		s    string
		dir  bidi.Direction
		want string
	}{
		{"a ١٢٣", bidi.LeftToRight, "[0 0 2 2 2]"},
		{"١٢٣ b", bidi.LeftToRight, "[2 2 2 0 0]"},
		{"ab שלום", bidi.LeftToRight, "[0 0 0 1 1 1 1]"},
		{"אב 12 c", bidi.RightToLeft, "[1 1 1 2 2 1 2]"},
		{"a (ב) c", bidi.LeftToRight, "[0 0 0 1 0 0 0]"},
	} {
		rs := p.Segment([]rune(c.s), c.dir)
		if got := fmt.Sprint(runLevels(&rs)); got != c.want {
			t.Errorf("%q: levels %s, want %s", c.s, got, c.want)
		}
	}
}
