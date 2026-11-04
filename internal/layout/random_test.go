package layout

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Randomized differential testing against Chromium.
//
// randomCases generates seeded random trees from the layout property pool.
// They go through the same pipeline as the hand-written cases: `make
// crosscheck` renders them in Chromium into testdata/layout/chrome.json,
// and TestLayoutCases compares. The generator is deterministic (PCG with a
// fixed seed per case), so the case list is stable across runs and
// machines. Changing the generator changes the cases: re-run `make
// crosscheck` afterwards.
//
// Mismatches that are deliberately not fixed are listed in
// knownRandomMismatches with the deviation number from spec/SUBSET.md.

const randomCaseCount = 2400

// knownRandomMismatches maps a random case name to the spec deviation
// that explains why it differs from Chromium.
var knownRandomMismatches = map[string]string{
	"random_015":  "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_114":  "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_416":  "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_957":  "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_1145": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_1579": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_1609": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_1787": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_2110": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_2131": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_2230": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_2249": "D11 (absolute box with top+bottom that does not fit its inset-modified containing block)",
	"random_117":  "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_218":  "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_460":  "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_525":  "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_1623": "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_2142": "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_2286": "D12 (intrinsic size of wrapping containers with explicit flex-basis items)",
	"random_1006": "D13 (cyclic percentages against content-derived definite sizes)",
	"random_1051": "D13 (cyclic percentages against content-derived definite sizes)",
	"random_2254": "D13 (cyclic percentages against content-derived definite sizes)",
	"random_1347": "D14 (sub-pixel rounding, at most 2/64 px)",
	"random_1465": "D14 (sub-pixel rounding, at most 2/64 px)",
	"random_1863": "D14 (sub-pixel rounding, at most 2/64 px)",
}

type gen struct {
	r   *rand.Rand
	ids int
}

func (g *gen) pick(opts ...string) string { return opts[g.r.IntN(len(opts))] }
func (g *gen) chance(p int) bool          { return g.r.IntN(100) < p }
func (g *gen) px(lo, hi int) string       { return fmt.Sprintf("%dpx", lo+g.r.IntN(hi-lo+1)) }

func (g *gen) length(auto bool) string {
	switch g.r.IntN(6) {
	case 0, 1, 2:
		return g.px(0, 120)
	case 3:
		return fmt.Sprintf("%d%%", 5+5*g.r.IntN(20))
	default:
		if auto {
			return "auto"
		}
		return g.px(0, 40)
	}
}

func (g *gen) css(depth int, isRoot bool) string {
	var d []string
	add := func(p int, f func() string) {
		if g.chance(p) {
			d = append(d, f())
		}
	}
	add(50, func() string { return "flex-direction:" + g.pick("row", "column", "row-reverse", "column-reverse") })
	add(25, func() string { return "flex-wrap:" + g.pick("wrap", "wrap-reverse", "nowrap") })
	add(30, func() string {
		return "justify-content:" + g.pick("flex-start", "flex-end", "center", "space-between", "space-around", "space-evenly")
	})
	add(30, func() string { return "align-items:" + g.pick("stretch", "flex-start", "flex-end", "center") })
	add(25, func() string {
		return "align-content:" + g.pick("stretch", "flex-start", "flex-end", "center", "space-between", "space-around", "space-evenly")
	})
	add(20, func() string { return "gap:" + g.px(0, 12) + " " + g.px(0, 12) })
	if !isRoot {
		add(25, func() string { return "align-self:" + g.pick("auto", "stretch", "flex-start", "flex-end", "center") })
		add(35, func() string { return fmt.Sprintf("flex-grow:%d", g.r.IntN(4)) })
		add(25, func() string { return fmt.Sprintf("flex-shrink:%d", g.r.IntN(4)) })
		add(25, func() string { return "flex-basis:" + g.length(true) })
		add(15, func() string { return "margin:" + g.pick("auto", g.px(-10, 15), g.px(0, 10)+" auto") })
		add(10, func() string {
			return "margin-" + g.pick("top", "right", "bottom", "left") + ":" + g.pick("auto", g.px(-8, 12), "5%")
		})
		add(8, func() string {
			s := "position:absolute"
			for _, side := range []string{"top", "right", "bottom", "left"} {
				if g.chance(45) {
					s += ";" + side + ":" + g.pick(g.px(-5, 30), "10%")
				}
			}
			return s
		})
		add(8, func() string {
			return "position:relative;" + g.pick("left", "right", "top", "bottom") + ":" + g.px(-10, 10)
		})
		add(4, func() string { return "display:none" })
	}
	add(35, func() string { return "width:" + g.length(true) })
	add(30, func() string { return "height:" + g.length(true) })
	add(12, func() string { return "min-width:" + g.length(false) })
	add(12, func() string { return "min-height:" + g.length(false) })
	add(12, func() string { return "max-width:" + g.pick(g.px(10, 150), "50%", "none") })
	add(12, func() string { return "max-height:" + g.pick(g.px(10, 150), "50%", "none") })
	add(25, func() string { return "padding:" + g.px(0, 10) + " " + g.pick(g.px(0, 10), "5%") })
	add(20, func() string { return "border-width:" + g.px(0, 6) + " " + g.px(0, 3) })
	add(15, func() string { return "box-sizing:border-box" })
	add(10, func() string { return "overflow:" + g.pick("hidden", "scroll") })
	return strings.Join(d, ";")
}

func (g *gen) node(depth int, isRoot bool) *spec {
	g.ids++
	id := fmt.Sprintf("n%d", g.ids)
	css := g.css(depth, isRoot)
	if !isRoot && (depth >= 3 || g.chance(35)) {
		if g.chance(50) {
			return F(id, css, 5+g.r.IntN(60), 5+g.r.IntN(40))
		}
		return T(id, css, 1+g.r.IntN(8), 5+g.r.IntN(20), 5+g.r.IntN(10))
	}
	n := E(id, css)
	for k := g.r.IntN(5); k > 0; k-- {
		n.kids = append(n.kids, g.node(depth+1, false))
	}
	return n
}

func randomCases() []layoutCase {
	var out []layoutCase
	for i := 0; i < randomCaseCount; i++ {
		g := &gen{r: rand.New(rand.NewPCG(0xF1E7, uint64(i)))}
		vw, vh := 100+g.r.IntN(300), 100+g.r.IntN(300)
		root := g.node(0, true)
		root.id = "root"
		out = append(out, layoutCase{name: fmt.Sprintf("random_%03d", i), vw: vw, vh: vh, root: root})
	}
	return out
}

func allCases() []layoutCase { return append(append([]layoutCase{}, cases...), randomCases()...) }
