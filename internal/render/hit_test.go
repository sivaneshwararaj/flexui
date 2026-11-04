package render

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// TestHitTestZOrder checks specific stacking cases.
func TestHitTestZOrder(t *testing.T) {
	d := Build(`<body>
<div id="a" style="position: absolute; left: 0; top: 0; width: 60px; height: 60px; z-index: 1">
  <div id="a1" style="position: absolute; left: 10px; top: 10px; width: 80px; height: 20px; z-index: 999"></div>
</div>
<div id="b" style="position: absolute; left: 40px; top: 0; width: 60px; height: 60px; z-index: 2"></div>
<div id="c" style="position: absolute; left: 0; top: 100px; width: 50px; height: 50px; background-color: red; z-index: 0">
  <div id="c1" style="position: absolute; left: 10px; top: 10px; width: 50px; height: 50px; z-index: -1"></div>
</div>
<div id="d" style="position: absolute; left: 100px; top: 100px; width: 40px; height: 40px; overflow: hidden">
  <div id="d1" style="position: absolute; left: 20px; top: 20px; width: 60px; height: 60px"></div>
</div>
<div id="e" style="position: absolute; left: 200px; top: 0; width: 50px; height: 50px"></div>
<div id="f" style="position: absolute; left: 220px; top: 20px; width: 50px; height: 50px">text here</div>
<div id="g" style="position: absolute; left: 200px; top: 100px; width: 50px; height: 50px; opacity: 0.5"></div>
<div id="h" style="position: absolute; left: 210px; top: 110px; width: 50px; height: 50px; z-index: -1"></div>
</body>`, "")
	d.Layout(300, 200)
	for _, c := range []struct {
		x, y int
		want string
	}{
		{50, 20, "b"},   // b (z 2) over a and over a's z-999 child: a's context is z 1
		{20, 20, "a1"},  // a1 within a's context
		{5, 5, "a"},     // only a
		{55, 115, "c1"}, // c1 (z -1 inside c's context) outside c
		{20, 120, "c1"}, // ...and above c's background (Appendix E step 3)
		{5, 105, "c"},
		{110, 110, "d"},    // d
		{130, 130, "d1"},   // d1 inside d's clip
		{150, 150, "body"}, // d1 outside d's clip: not hit
		{230, 30, "f"},     // later sibling on top (tree order)
		{210, 10, "e"},
		{235, 25, "f"},     // text belongs to f
		{255, 135, "body"}, // h (z -1) is in the canvas context: behind body
		{215, 115, "g"},    // g (opacity context, z 0) above h (z -1)
		{299, 199, "body"},
	} {
		b := d.HitTest(geom.Px(c.x), geom.Px(c.y))
		got := "nil"
		if b != nil {
			got = b.El.ID()
			if got == "" {
				got = b.El.Tag()
			}
		}
		if got != c.want {
			t.Errorf("(%d,%d): hit %s, want %s", c.x, c.y, got, c.want)
		}
	}
}

// TestHitTestMatchesPaint renders random overlapping boxes, each with a
// unique opaque color, and checks at every pixel center (away from
// edges) that hit testing returns the element whose color is visible
// there: hit order equals paint order, including z-index, opacity
// contexts and overflow clips.
func TestHitTestMatchesPaint(t *testing.T) {
	checked := map[string]int{}
	for seed := 0; seed < 40; seed++ {
		r := rand.New(rand.NewPCG(uint64(seed), 99))
		var b strings.Builder
		var colors []string
		n := 0
		var box func(depth int)
		box = func(depth int) {
			n++
			id := n
			c := fmt.Sprintf("#%02x%02x%02x", 20+id*7%200, 30+id*13%200, 40+id*29%200)
			colors = append(colors, c)
			css := fmt.Sprintf("position: absolute; left: %dpx; top: %dpx; width: %dpx; height: %dpx; background-color: %s",
				r.IntN(120), r.IntN(80), 20+r.IntN(100), 20+r.IntN(80), c)
			if r.IntN(2) == 0 {
				css += fmt.Sprintf("; z-index: %d", r.IntN(5)-2)
			}
			if r.IntN(4) == 0 {
				css += "; overflow: hidden"
			}
			if r.IntN(5) == 0 {
				css += "; opacity: 0.99"
			}
			fmt.Fprintf(&b, `<div id="e%d" style="%s">`, id, css)
			if depth > 0 {
				for k := r.IntN(3); k > 0; k-- {
					box(depth - 1)
				}
			}
			b.WriteString("</div>")
		}
		for k := 0; k < 4; k++ {
			box(2)
		}
		src := `<body style="background-color: #fff">` + b.String() + "</body>"
		d := Build(src, "")
		d.Layout(240, 160)
		img, _ := Render(src, "", 240, 160, 1)
		for y := 0; y < 160; y += 3 {
			for x := 0; x < 240; x += 3 {
				p := img.RGBAAt(x, y)
				// Skip pixels whose color is not one element's exactly
				// (opacity 0.99 blends slightly) and edge pixels.
				owner := ""
				for i, c := range colors {
					if fmt.Sprintf("#%02x%02x%02x", p.R, p.G, p.B) == c {
						owner = fmt.Sprintf("e%d", i+1)
					}
				}
				if p == (img.RGBAAt(0, 0)) && owner == "" {
					owner = "body"
				}
				if owner == "" {
					continue
				}
				hb := d.HitTest(geom.Px(x)+32, geom.Px(y)+32)
				got := "body"
				if hb != nil && hb.El.ID() != "" {
					got = hb.El.ID()
				}
				if got == "body" {
					checked["body"]++
				} else {
					checked["element"]++
				}
				if got != owner {
					t.Fatalf("seed %d (%d,%d): painted %s, hit %s\n%s", seed, x, y, owner, got, src)
				}
			}
		}
	}
	t.Logf("checked %d element pixels and %d body pixels", checked["element"], checked["body"])
	if checked["element"] < 40000 {
		t.Errorf("too few pixels checked: %v", checked)
	}
}
