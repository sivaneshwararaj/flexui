package main

import (
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sivaneshwararaj/flexui/platform/headless"
)

var update = flag.Bool("update", false, "rewrite the goldens")

// TestDemo runs the scripted session and compares every state with its
// golden PNG exactly (testdata/<state>.png), and checks the app state
// along the way.
func TestDemo(t *testing.T) {
	d := NewDemo()
	checks := map[string]func(){
		"2-search": func() {
			if n := len(d.Visible()); n != 4 {
				t.Errorf("search: %d visible contacts, want 4", n)
			}
		},
		"3-selected": func() {
			if d.Draft().Name != "Margaret Hamilton" {
				t.Errorf("selected %q", d.Draft().Name)
			}
		},
		"5-saved": func() {
			if c := d.contacts[d.find(d.selected)]; c.Email != "margaret@apollo.nasa.gov" || d.Dirty() {
				t.Errorf("after save: %+v dirty %v", c, d.Dirty())
			}
		},
		"6-dialog": func() {
			if !d.Confirm() {
				t.Error("dialog not open")
			}
		},
		"7-deleted": func() {
			if d.Count() != 13 || d.Confirm() || d.Draft().Name != "John McCarthy" {
				t.Errorf("after delete: %d contacts, dialog %v, selected %q", d.Count(), d.Confirm(), d.Draft().Name)
			}
		},
		"8-added": func() {
			if d.Count() != 14 || d.Draft().Name != "Sophie Wilson" || d.Draft().Email != "sophie@acorn.co.uk" {
				t.Errorf("after add: %d contacts, draft %+v", d.Count(), d.Draft())
			}
		},
	}
	snapHook = func(name string, s *headless.Session) {
		if c := checks[name]; c != nil {
			c()
		}
		golden(t, name, s.Frame())
	}
	defer func() { snapHook = nil }()
	names := Script(d, headless.Start(d.App))
	if len(names) != 8 {
		t.Errorf("%d states", len(names))
	}
	if w := d.App.Warnings(); len(w) != 0 {
		t.Errorf("warnings: %q", w)
	}
}

// TestEscapeAndBackdrop: the dialog closes with Escape and with a click on
// the backdrop, but not with a click inside it.
func TestEscapeAndBackdrop(t *testing.T) {
	d := NewDemo()
	s := headless.Start(d.App)
	s.Click(center(d.App, "delete"))
	x, y, _, _, _ := d.App.Bounds("cancel")
	s.Click(x-60, y-30) // inside the dialog, not on a button
	if !d.Confirm() {
		t.Fatal("click inside the dialog closed it")
	}
	s.Click(5, 5) // the backdrop
	if d.Confirm() {
		t.Fatal("backdrop click did not close the dialog")
	}
	s.Click(center(d.App, "delete"))
	s.Key("Escape", 0)
	if d.Confirm() || d.Count() != 14 {
		t.Fatalf("Escape: dialog %v, %d contacts", d.Confirm(), d.Count())
	}
}

func golden(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	path := filepath.Join("testdata", name+".png")
	if *update {
		os.MkdirAll("testdata", 0o755)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, img)
		f.Close()
		return
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	defer f.Close()
	want, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if want.Bounds() != img.Bounds() {
		t.Fatalf("%s: size %v, want %v", name, img.Bounds(), want.Bounds())
	}
	n := 0
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			r, g, b, a := want.At(x, y).RGBA()
			p := img.RGBAAt(x, y)
			if p.R != uint8(r>>8) || p.G != uint8(g>>8) || p.B != uint8(b>>8) || p.A != uint8(a>>8) {
				n++
			}
		}
	}
	if n > 0 {
		os.MkdirAll(filepath.Join("testdata", "_failed"), 0o755)
		f, _ := os.Create(filepath.Join("testdata", "_failed", name+".png"))
		png.Encode(f, img)
		f.Close()
		t.Errorf("%s: %d pixels differ from the golden", name, n)
	}
}

// TestStatusClearedOnSelect is a regression test: the "Saved ..." line
// used to stay when another contact was selected.
func TestStatusClearedOnSelect(t *testing.T) {
	d := NewDemo()
	s := headless.Start(d.App)
	s.Click(center(d.App, "notes"))
	s.Type(" Edited.")
	s.Click(center(d.App, "save"))
	if d.Status() != "Saved Ada Lovelace." {
		t.Fatalf("status after save: %q", d.Status())
	}
	if _, _, _, _, ok := d.App.Bounds("status"); !ok {
		t.Fatal("status line not shown after save")
	}
	// Select the second contact in the list.
	_, ly, _, _, _ := d.App.Bounds("list")
	lx, _ := center(d.App, "list")
	s.Click(lx, ly+1+49+24)
	if d.Draft().Name != "Alan Turing" {
		t.Fatalf("selected %q", d.Draft().Name)
	}
	if d.Status() != "" {
		t.Errorf("status after selecting another contact: %q", d.Status())
	}
	if _, _, _, _, ok := d.App.Bounds("status"); ok {
		t.Error("status line still shown after selecting another contact")
	}
	// Revert also clears it; delete and add keep their own messages.
	s.Click(center(d.App, "delete"))
	s.Click(center(d.App, "confirm"))
	if d.Status() != "Deleted Alan Turing." {
		t.Errorf("status after delete: %q", d.Status())
	}
}
