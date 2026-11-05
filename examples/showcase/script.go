package main

import "github.com/sivaneshwararaj/flexui/platform/headless"

// snapHook is called after each step of Script (main writes PNGs, the
// test compares goldens).
var snapHook = func(step string, s *headless.Session) {}

// click clicks the center of the element with the given id.
func click(sc *Showcase, s *headless.Session, id string) {
	x, y, w, h, ok := sc.App.Bounds(id)
	if !ok {
		panic("no element #" + id)
	}
	s.Click(x+w/2, y+h/2)
}

// Script drives a short session through every tab.
func Script(sc *Showcase, s *headless.Session) {
	snapHook("1-order", s)
	// Multi-select: three products, more of one, a filter, a coupon.
	click(sc, s, "p1")
	click(sc, s, "p4")
	click(sc, s, "inc4")
	click(sc, s, "inc4")
	click(sc, s, "p2")
	click(sc, s, "tax18")
	click(sc, s, "coupon")
	s.Type("save10")
	snapHook("2-selected", s)
	click(sc, s, "cat-Dairy")
	click(sc, s, "all")
	snapHook("3-filtered", s)
	click(sc, s, "checkout")
	snapHook("4-checkout", s)
	click(sc, s, "place")
	snapHook("5-placed", s)
	// Calculator: 12.5 × 8 − 4 = and keyboard input.
	click(sc, s, "tab-calc")
	for _, k := range []string{"k1", "k2", "k.", "k5", "k×", "k8", "k−", "k4", "k="} {
		click(sc, s, k)
	}
	snapHook("6-calc", s)
	click(sc, s, "calc")
	for _, k := range []string{"Escape", "7", "/", "2", "Enter"} {
		s.Key(k, 0) // keys as a keyboard sends them: 7 ÷ 2 = 3.5
	}
	snapHook("7-calc-keys", s)
	click(sc, s, "tab-lang")
	click(sc, s, "typed")
	s.Type("தமிழ் مرحبا Hello")
	snapHook("8-languages", s)
	click(sc, s, "tab-layout")
	snapHook("9-layout", s)
}
