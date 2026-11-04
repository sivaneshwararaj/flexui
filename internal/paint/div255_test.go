package paint

import "testing"

// SUBSET.md §6.3 claims div255(x) == round(x/255) (halves up) for all
// 0 <= x <= 255*255, the largest value any call site passes (both blend
// terms sum to at most 255*255). It is NOT exact much beyond that
// (div255(65663) is off by one).
func TestDiv255Exact(t *testing.T) {
	for x := uint32(0); x <= 255*255; x++ {
		if got, want := div255(x), (2*x+255)/510; got != want {
			t.Fatalf("div255(%d) = %d, want %d", x, got, want)
		}
	}
}
