package flexui

import (
	"os"
	"testing"
)

// TestMain turns on the damage check: every frame repainted only in its
// damaged rectangles is also repainted in full and must be byte-identical
// (internal/view, FLEXUI_CHECK_DAMAGE).
func TestMain(m *testing.M) {
	os.Setenv("FLEXUI_CHECK_DAMAGE", "1")
	os.Exit(m.Run())
}
