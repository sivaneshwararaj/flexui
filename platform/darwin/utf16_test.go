package darwin

import "testing"

func TestUTF16ToRune(t *testing.T) {
	s := "かん👍x"
	for _, c := range []struct{ u, want int }{{0, 0}, {1, 1}, {2, 2}, {4, 3}, {5, 4}, {9, 4}} {
		if got := utf16ToRune(s, c.u); got != c.want {
			t.Errorf("utf16ToRune(%d) = %d, want %d", c.u, got, c.want)
		}
	}
	if utf16Len(s) != 5 {
		t.Errorf("utf16Len %d", utf16Len(s))
	}
}
