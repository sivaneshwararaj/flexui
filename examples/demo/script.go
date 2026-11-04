package main

import (
	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// center returns the center of the element with the given id.
func center(a *flexui.App, id string) (float64, float64) {
	x, y, w, h, ok := a.Bounds(id)
	if !ok {
		panic("demo script: no element #" + id)
	}
	return x + w/2, y + h/2
}

// Script drives the demo through a scripted session, calling snap after
// each interesting state; it returns the names of the states. The same
// script produces the goldens (demo_test.go) and the PNGs of `go run`.
func Script(d *Demo, s *headless.Session) []string {
	var names []string
	snap := func(name string) {
		s.Leave() // no hover in snapshots
		names = append(names, name)
		if snapHook != nil {
			snapHook(name, s)
		}
	}
	snap("1-start")

	// Search the list.
	s.Click(center(d.App, "search"))
	s.Type("ar")
	snap("2-search")

	// Pick Margaret Hamilton from the filtered list (the fourth match;
	// rows are 49px: 8+8 padding, 32px content, 1px border).
	_, ly, _, _, _ := d.App.Bounds("list")
	lx, _ := center(d.App, "list")
	s.Click(lx, ly+1+3*49+24)
	snap("3-selected")

	// Edit the email and save.
	s.Click(center(d.App, "email"))
	s.Key("a", platform.ModMeta)
	s.Type("margaret@apollo.nasa.gov")
	snap("4-edited")
	s.Click(center(d.App, "save"))
	snap("5-saved")

	// Delete: the dialog, Escape closes it, then delete for real.
	s.Click(center(d.App, "delete"))
	snap("6-dialog")
	s.Key("Escape", 0)
	s.Click(center(d.App, "delete"))
	s.Click(center(d.App, "confirm"))
	snap("7-deleted")

	// Clear the search and add a contact; the name field is focused.
	s.Click(center(d.App, "search"))
	s.Key("a", platform.ModMeta)
	s.Key("Backspace", 0)
	s.Click(center(d.App, "add"))
	s.Key("a", platform.ModMeta)
	s.Type("Sophie Wilson")
	s.Key("Tab", 0)
	s.Type("sophie@acorn.co.uk")
	s.Click(center(d.App, "save"))
	snap("8-added")
	return names
}

// snapHook is set by tests to check each state.
var snapHook func(name string, s *headless.Session)
