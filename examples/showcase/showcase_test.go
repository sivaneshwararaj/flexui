package main

import (
	"bytes"
	"flag"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sivaneshwararaj/flexui/platform/headless"
)

var update = flag.Bool("update", false, "rewrite the goldens")

// TestMain turns on flexui's damage check: every incremental frame is
// compared with a full repaint.
func TestMain(m *testing.M) {
	os.Setenv("FLEXUI_CHECK_DAMAGE", "1")
	os.Exit(m.Run())
}

// TestShowcase runs the scripted session, compares every screen with its
// golden PNG pixel for pixel (testdata/<step>.png) and checks the computed
// values along the way.
func TestShowcase(t *testing.T) {
	sc := NewShowcase()
	checks := map[string]func(){
		"2-selected": func() {
			// Apples 3.49 + 3 × sourdough 4.50 + bananas 1.99 = 18.98;
			// SAVE10: -1.90; 18% tax on 17.08 = 3.07; total 20.15.
			if sc.Count() != 5 || money(sc.Subtotal()) != "$18.98" || money(sc.Discount()) != "$1.90" ||
				money(sc.Tax()) != "$3.07" || money(sc.Total()) != "$20.15" {
				t.Errorf("totals: %d items, %s %s %s %s", sc.Count(), money(sc.Subtotal()), money(sc.Discount()), money(sc.Tax()), money(sc.Total()))
			}
		},
		"3-filtered": func() {
			if n := len(sc.Products()); n != 3 {
				t.Errorf("Dairy filter shows %d products, want 3", n)
			}
			if sc.Count() != 8 { // select all adds the three dairy products
				t.Errorf("after select all: %d items", sc.Count())
			}
		},
		"4-checkout": func() {
			if !sc.Checkout() {
				t.Error("checkout dialog not open")
			}
		},
		"5-placed": func() {
			if sc.Checkout() || sc.Count() != 0 || sc.Placed() == "" {
				t.Errorf("after placing: dialog %v, %d items, %q", sc.Checkout(), sc.Count(), sc.Placed())
			}
		},
		"6-calc": func() {
			if sc.Display() != "96" { // 12.5 × 8 − 4
				t.Errorf("calculator shows %q, want 96", sc.Display())
			}
		},
		"7-calc-keys": func() {
			if sc.Display() != "3.5" { // 7 / 2 from the keyboard
				t.Errorf("calculator shows %q, want 3.5", sc.Display())
			}
		},
		"8-languages": func() {
			if sc.App.Value("typed") != "தமிழ் مرحبا Hello" {
				t.Errorf("typed %q", sc.App.Value("typed"))
			}
		},
	}
	snapHook = func(step string, s *headless.Session) {
		if w := sc.App.Warnings(); len(w) > 0 {
			t.Fatalf("%s: warnings %q", step, w)
		}
		if f := checks[step]; f != nil {
			f()
		}
		var b bytes.Buffer
		if err := png.Encode(&b, s.Frame()); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", step+".png")
		if *update {
			os.MkdirAll("testdata", 0o755)
			if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run with -update)", err)
		}
		// Compare pixels, not PNG bytes: Go's PNG/zlib encoder output
		// differs between Go versions for the same image.
		m, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		want := image.NewRGBA(m.Bounds())
		draw.Draw(want, want.Rect, m, m.Bounds().Min, draw.Src)
		got := s.Frame()
		if !want.Rect.Eq(got.Rect) || !bytes.Equal(want.Pix, got.Pix) {
			os.MkdirAll(filepath.Join("testdata", "_failed"), 0o755)
			os.WriteFile(filepath.Join("testdata", "_failed", step+".png"), b.Bytes(), 0o644)
			t.Errorf("%s differs from its golden (written to testdata/_failed)", step)
		}
	}
	Script(sc, headless.Start(sc.App))
}

// TestCalculator checks the calculator logic on its own.
func TestCalculator(t *testing.T) {
	for _, c := range []struct{ keys, want string }{
		{"1+2=", "3"}, {"12.5×8−4=", "96"}, {"7÷2=", "3.5"}, {"5÷0=", "0"},
		{"50%", "0.5"}, {"123⌫⌫", "1"}, {"9C", "0"}, {"2+3×4=", "20"}, // left to right
	} {
		s := &Showcase{display: "0", fresh: true}
		for _, k := range c.keys {
			s.press(string(k))
		}
		if s.display != c.want {
			t.Errorf("%s: %q, want %q", c.keys, s.display, c.want)
		}
	}
}
