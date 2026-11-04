package main

import (
	"os"
	"testing"
)

// TestMain turns on flexui's damage check for the tests (every
// incremental frame is compared with a full repaint), but not for the
// benchmarks, which measure the real cost.
func TestMain(m *testing.M) {
	bench := false
	for _, a := range os.Args {
		if len(a) > 11 && a[:12] == "-test.bench=" {
			bench = true
		}
	}
	if !bench {
		os.Setenv("FLEXUI_CHECK_DAMAGE", "1")
	}
	os.Exit(m.Run())
}
