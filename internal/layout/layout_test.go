package layout

import (
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Re-running layout on the same tree must give identical results (the
// per-node cache is reset between passes).
func TestLayoutIdempotent(t *testing.T) {
	for _, c := range cases {
		root := build(t, c.root)
		Layout(root, geom.Px(c.vw), geom.Px(c.vh))
		first := map[string]rect{}
		collect(c.root, root, 0, 0, first)
		Layout(root, geom.Px(c.vw+10), geom.Px(c.vh+10))
		Layout(root, geom.Px(c.vw), geom.Px(c.vh))
		second := map[string]rect{}
		collect(c.root, root, 0, 0, second)
		for id, r := range first {
			if second[id] != r {
				t.Errorf("%s/%s: %v then %v", c.name, id, r, second[id])
			}
		}
	}
}

func TestBoxDetails(t *testing.T) {
	root := build(t, E("root", "padding:1px 2px 3px 4px;border-width:5px;flex-direction:row",
		E("a", "margin:1px 2px 3px auto;width:10px;padding:2px;border-width:1px")))
	Layout(root, geom.Px(100), geom.Px(50))
	rb := root.Box
	if rb.Padding != (geom.Edges{Top: geom.Px(1), Right: geom.Px(2), Bottom: geom.Px(3), Left: geom.Px(4)}) {
		t.Errorf("root padding %+v", rb.Padding)
	}
	if rb.Border.Left != geom.Px(5) {
		t.Errorf("root border %+v", rb.Border)
	}
	a := root.Children[0].Box
	// Content box of root: x from 9 to 93 (100-5-2); a is 16px wide incl.
	// padding and border, right margin 2 -> x = 93-2-16 = 75. The auto
	// left margin absorbs the rest: 75-9 = 66.
	if a.X != geom.Px(75) || a.W != geom.Px(16) || a.Margin.Left != geom.Px(66) {
		t.Errorf("a box %+v", a)
	}
}

func TestDisplayNoneZeroBox(t *testing.T) {
	root := build(t, E("root", "", E("a", "display:none;width:10px;height:10px")))
	root.Children[0].Box = Box{X: 99, W: 99}
	Layout(root, geom.Px(100), geom.Px(50))
	if root.Children[0].Box != (Box{}) {
		t.Errorf("display:none box %+v", root.Children[0].Box)
	}
}

func TestContentExtent(t *testing.T) {
	root := build(t, E("root", "width:100px;height:50px;padding:5px;border-width:1px;overflow:scroll",
		E("a", "height:80px;flex-shrink:0;margin-bottom:3px"),
		E("abs", "position:absolute;left:150px;top:0;width:10px;height:10px")))
	Layout(root, geom.Px(300), geom.Px(300))
	b := root.Box
	// padding box is 110x60. In-flow child ends at 5+80+3 = 88, plus
	// bottom padding 5 = 93. The abs child ends at 160 horizontally.
	if b.ContentW != geom.Px(160) || b.ContentH != geom.Px(93) {
		t.Errorf("content extent %v x %v", b.ContentW, b.ContentH)
	}
}

// Deeply nested auto-sized containers must not blow up exponentially.
func TestDeepNestingCost(t *testing.T) {
	calls := 0
	leafStyle := style.Initial()
	var mk func(depth int) *Node
	mk = func(depth int) *Node {
		st := style.Initial()
		if depth%2 == 0 {
			st.FlexDirection = style.Row
		}
		st.AlignItems = style.AlignCenter
		st.Padding = [4]style.Length{style.Px(1), style.Px(1), style.Px(1), style.Px(1)}
		n := &Node{Style: &st}
		if depth == 0 {
			return &Node{Style: &leafStyle, Measure: countingMeasure{textMeasure{5, geom.Px(10), geom.Px(10)}, &calls}}
		}
		n.Children = []*Node{mk(depth - 1), mk(depth - 1)}
		return n
	}
	root := mk(10) // 1024 leaves
	Layout(root, geom.Px(800), geom.Px(600))
	if calls > 1024*16 {
		t.Errorf("measure called %d times for 1024 leaves", calls)
	}
	t.Logf("measure calls: %d for 1024 leaves", calls)
}

func BenchmarkLayoutCases(b *testing.B) {
	roots := make([]*Node, len(cases))
	for i, c := range cases {
		roots[i] = build(b, c.root)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, c := range cases {
			Layout(roots[j], geom.Px(c.vw), geom.Px(c.vh))
		}
	}
}

type countingMeasure struct {
	textMeasure
	calls *int
}

func (m countingMeasure) Widths() (geom.Unit, geom.Unit) {
	*m.calls++
	return m.textMeasure.Widths()
}

func (m countingMeasure) Height(w geom.Unit) geom.Unit {
	*m.calls++
	return m.textMeasure.Height(w)
}
