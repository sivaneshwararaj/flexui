package geom

import "testing"

func TestUnitString(t *testing.T) {
	for _, c := range []struct {
		u    Unit
		want string
	}{
		{Px(12), "12px"},
		{Px(12) + 32, "12.5px"},
		{1, "0.015625px"},
		{-Px(3), "-3px"},
	} {
		if got := c.u.String(); got != c.want {
			t.Errorf("%d: got %q want %q", int64(c.u), got, c.want)
		}
	}
}

func TestMulDiv(t *testing.T) {
	for _, c := range []struct{ a, b, c, want int64 }{
		{10, 3, 4, 7},
		{-10, 3, 4, -7},
		{1 << 40, 1 << 40, 1 << 41, 1 << 39},
		{0, 5, 7, 0},
	} {
		if got := MulDiv(c.a, c.b, c.c); got != c.want {
			t.Errorf("MulDiv(%d,%d,%d)=%d want %d", c.a, c.b, c.c, got, c.want)
		}
	}
}
