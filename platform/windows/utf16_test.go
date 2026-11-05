package windows

import (
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/testtext"
)

func TestClipTextRoundTrip(t *testing.T) {
	for _, smp := range testtext.Samples {
		u, err := encodeClipText(smp.Text)
		if err != nil {
			t.Fatal(smp.Name, err)
		}
		if u[len(u)-1] != 0 {
			t.Errorf("%s: not NUL-terminated", smp.Name)
		}
		for _, c := range u[:len(u)-1] {
			if c == 0 {
				t.Errorf("%s: NUL inside", smp.Name)
			}
		}
		if got := decodeClipText(u); got != smp.Text {
			t.Errorf("%s: round trip %+q, want %+q", smp.Name, got, smp.Text)
		}
		// Without the terminator (the block ends exactly at the text).
		if got := decodeClipText(u[:len(u)-1]); got != smp.Text {
			t.Errorf("%s: unterminated %+q", smp.Name, got)
		}
		// Bytes after the NUL (GlobalAlloc rounds sizes up) are ignored.
		if got := decodeClipText(append(append([]uint16{}, u...), 'x', 'y')); got != smp.Text {
			t.Errorf("%s: trailing bytes %+q", smp.Name, got)
		}
	}
}

func TestClipTextSurrogates(t *testing.T) {
	u, _ := encodeClipText("a👍b")
	want := []uint16{'a', 0xD83D, 0xDC4D, 'b', 0}
	if len(u) != len(want) {
		t.Fatalf("%x", u)
	}
	for i := range u {
		if u[i] != want[i] {
			t.Fatalf("%x, want %x", u, want)
		}
	}
	// A lone surrogate from a misbehaving app becomes U+FFFD, not garbage.
	if got := decodeClipText([]uint16{'a', 0xD83D, 'b', 0}); got != "a�b" {
		t.Errorf("lone surrogate: %+q", got)
	}
	if _, err := encodeClipText("a\x00b"); err == nil {
		t.Error("NUL accepted")
	}
}

func TestIMEText(t *testing.T) {
	u, _ := encodeClipText("かん字👍x")
	u = u[:len(u)-1] // no terminator from ImmGetCompositionStringW
	for _, c := range []struct{ cursor, want int }{{0, 0}, {2, 2}, {3, 3}, {5, 4}, {6, 5}, {99, 5}} {
		s, r := imeText(u, c.cursor)
		if s != "かん字👍x" || r != c.want {
			t.Errorf("cursor %d: %q rune %d, want %d", c.cursor, s, r, c.want)
		}
	}
}
