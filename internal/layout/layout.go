// Package layout is the flexbox layout engine.
//
// Input is a tree of *Node, each with a computed *style.Style, children,
// and (for leaves such as text and images) an optional measure callback.
// Output is written to each Node's Box. All math is integer geom.Unit
// (1/64 px), so results are identical on every platform.
//
// The algorithm follows CSS Flexible Box Layout Level 1 §9 with the
// deviations listed in spec/SUBSET.md.
package layout

import (
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// AvailKind describes available space along one axis.
type AvailKind uint8

const (
	// Definite means a known amount of space (Avail.V).
	Definite AvailKind = iota
	// MinContent asks for the min-content size.
	MinContent
	// MaxContent asks for the max-content size.
	MaxContent
)

// Avail is the space available to a box along one axis.
type Avail struct {
	Kind AvailKind
	V    geom.Unit // only meaningful for Definite
}

// Opt is an optional length.
type Opt struct {
	V  geom.Unit
	OK bool
}

func some(v geom.Unit) Opt { return Opt{v, true} }

// MeasureFunc reports the content-box size of a leaf. known sizes, if set,
// are the final content-box sizes for that axis; avail describes the
// space available to the content box. The engine asks for widths with
// availW = MinContent or MaxContent, and for heights with a known width.
// Implementations must be deterministic and should be cheap.
type MeasureFunc func(knownW, knownH Opt, availW, availH Avail) geom.Size

// Box is the layout result for one node.
type Box struct {
	// X, Y: border-box position relative to the parent's border-box origin
	// (including relative-position offsets). W, H: border-box size.
	X, Y, W, H geom.Unit
	// Used margins (auto margins resolved), borders and padding.
	Margin, Border, Padding geom.Edges
	// ContentW, ContentH: size of the scrollable overflow area, measured
	// from the padding-box origin. At least the padding-box size.
	ContentW, ContentH geom.Unit
}

// Node is one box in the layout tree.
type Node struct {
	Style    *style.Style
	Children []*Node
	// Measure sizes leaf content (text, images). It is ignored for nodes
	// with children. A leaf without Measure has zero content size.
	Measure MeasureFunc
	// Box is written by Layout.
	Box Box

	cache  [cacheSize]cacheEntry
	ncache uint8
	next   uint8
}

const cacheSize = 12

type input struct {
	known [2]Opt // border-box size imposed by the parent
	// definite reports whether known[a] counts as definite for the
	// percentages of descendants (CSS Flexbox §9.8).
	definite [2]bool
	// ignoreSize treats the node's own width/height as auto on that axis,
	// to measure its content size (automatic minimum size, flex-basis:
	// content).
	ignoreSize [2]bool
	parent     [2]Opt   // containing block content size, for percentages
	avail      [2]Avail // space available for the border box
	perform    bool     // do final layout of descendants
	// wantW: the caller only needs the width. Heights are skipped where
	// possible, which keeps intrinsic sizing of deep trees cheap.
	wantW bool
}

type cacheEntry struct {
	in   input
	size geom.Size
}

// Layout lays out the tree rooted at root in a viewport of the given size.
// The root's containing block is the viewport, and an auto width or height
// fills the viewport (spec/SUBSET.md §2.6).
func Layout(root *Node, viewportW, viewportH geom.Unit) {
	resetCache(root)
	vp := [2]Opt{some(viewportW), some(viewportH)}
	view := [2]geom.Unit{viewportW, viewportH}
	b := resolveBox(root.Style, vp)
	var sz [2]geom.Unit
	for a := 0; a < 2; a++ {
		if b.size[a].OK {
			sz[a] = b.clamp(a, b.size[a].V)
		} else {
			s, e := axisSides(a)
			sz[a] = b.clamp(a, view[a]-b.margin[s]-b.margin[e])
		}
	}
	dx, dy := relativeOffset(root.Style, vp)
	root.Box = Box{
		X: b.margin[style.Left] + dx, Y: b.margin[style.Top] + dy,
		W: sz[0], H: sz[1],
		Margin: edges(b.margin),
	}
	root.compute(input{
		known:    [2]Opt{some(sz[0]), some(sz[1])},
		definite: [2]bool{true, true},
		parent:   vp,
		avail:    [2]Avail{{Definite, sz[0]}, {Definite, sz[1]}},
		perform:  true,
	})
}

func resetCache(n *Node) {
	n.ncache, n.next = 0, 0
	for _, c := range n.Children {
		resetCache(c)
	}
}

// compute returns the border-box size of n. With in.perform it also lays
// out n's descendants and fills n.Box's Border, Padding and Content
// fields (the parent sets position, size and margins). With in.wantW only
// the returned width is meaningful.
func (n *Node) compute(in input) geom.Size {
	if !in.perform {
		in = n.canonical(in)
		if in.known[0].OK && (in.wantW || in.known[1].OK) {
			return geom.Size{W: in.known[0].V, H: in.known[1].V}
		}
		for i := uint8(0); i < n.ncache; i++ {
			if n.cache[i].in == in {
				return n.cache[i].size
			}
		}
	}
	var s geom.Size
	if len(n.Children) == 0 {
		s = n.leaf(in)
	} else {
		s = n.flex(in)
	}
	if !in.perform {
		n.cache[n.next] = cacheEntry{in, s}
		n.next = (n.next + 1) % cacheSize
		if n.ncache < cacheSize {
			n.ncache++
		}
	}
	return s
}

// canonical drops inputs that cannot affect the result of a sizing call,
// so that equivalent calls share cache entries:
//   - The available height never matters: heights are content-based
//     (only a column-wrap container breaks lines by height, and it uses
//     its own height/max-height for that).
//   - For width-only calls, nothing about the height matters.
//
// Column-wrap containers are exempt (their width depends on their height).
// Known limitation: a column-wrap container nested inside a node sized by
// a width-only call sees no definite ancestor height for percentages.
func (n *Node) canonical(in input) input {
	st := n.Style
	if len(n.Children) > 0 && !st.FlexDirection.IsRow() && st.FlexWrap != style.NoWrap {
		in.wantW = false
		return in
	}
	in.avail[1] = Avail{Kind: MaxContent}
	if in.wantW {
		in.known[1], in.definite[1], in.ignoreSize[1], in.parent[1] = Opt{}, false, false, Opt{}
	}
	return in
}

func (n *Node) leaf(in input) geom.Size {
	b := resolveBox(n.Style, in.parent)
	sz, _ := ownSizes(&b, in)
	if (!sz[0].OK || !sz[1].OK) && n.Measure != nil {
		var kh Opt
		if sz[1].OK {
			kh = some(geom.Max(0, sz[1].V-b.pb[1]))
		}
		anyH := Avail{Kind: MaxContent}
		if !sz[0].OK {
			// Width: fit-content within the available space.
			var w geom.Unit
			av := innerAvail(in.avail[0], b.pb[0])
			if av.Kind == Definite {
				w = n.Measure(Opt{}, kh, Avail{Kind: MaxContent}, anyH).W
				if w > av.V {
					minc := n.Measure(Opt{}, kh, Avail{Kind: MinContent}, anyH).W
					w = geom.Max(minc, av.V)
				}
			} else {
				w = n.Measure(Opt{}, kh, av, anyH).W
			}
			sz[0] = some(b.clampIf(0, w+b.pb[0], in))
		}
		if !sz[1].OK && !in.wantW {
			kw := geom.Max(0, sz[0].V-b.pb[0])
			h := n.Measure(some(kw), Opt{}, Avail{Definite, kw}, anyH).H
			sz[1] = some(b.clampIf(1, h+b.pb[1], in))
		}
	}
	for a := 0; a < 2; a++ {
		if !sz[a].OK {
			sz[a] = some(b.clampIf(a, b.pb[a], in))
		}
	}
	if in.perform {
		n.Box.Border = edges(b.border)
		n.Box.Padding = edges(b.padding)
		n.Box.ContentW = sz[0].V - b.border[style.Left] - b.border[style.Right]
		n.Box.ContentH = sz[1].V - b.border[style.Top] - b.border[style.Bottom]
	}
	return geom.Size{W: sz[0].V, H: sz[1].V}
}

// ownSizes returns the node's border-box sizes that are known before
// looking at content (imposed by the parent or set in style), and whether
// each counts as definite for percentages.
func ownSizes(b *boxStyle, in input) (sz [2]Opt, definite [2]bool) {
	for a := 0; a < 2; a++ {
		if in.known[a].OK {
			sz[a], definite[a] = in.known[a], in.definite[a]
		} else if b.size[a].OK && !in.ignoreSize[a] {
			sz[a], definite[a] = some(b.clamp(a, b.size[a].V)), true
		}
	}
	return sz, definite
}

// innerAvail converts the space available to a border box into the space
// available to its content box. Max sizes are applied afterwards by
// clamping (fit-content = clamp(min(max-content, available))).
func innerAvail(a Avail, pb geom.Unit) Avail {
	if a.Kind == Definite {
		a.V = geom.Max(0, a.V-pb)
	}
	return a
}

// boxStyle holds a node's style resolved against its containing block.
// Sizes are border-box sizes.
type boxStyle struct {
	margin  [4]geom.Unit
	mAuto   [4]bool
	padding [4]geom.Unit
	border  [4]geom.Unit
	pb      [2]geom.Unit // padding+border per axis (0 = horizontal)
	size    [2]Opt
	min     [2]geom.Unit // at least pb
	minAuto [2]bool
	max     [2]Opt
}

func resolveBox(st *style.Style, cb [2]Opt) boxStyle {
	var b boxStyle
	for i := 0; i < 4; i++ {
		if st.Margin[i].IsAuto() {
			b.mAuto[i] = true
		} else {
			b.margin[i], _ = st.Margin[i].Resolve(cb[0].V, cb[0].OK)
		}
		b.padding[i], _ = st.Padding[i].Resolve(cb[0].V, cb[0].OK)
		b.border[i] = st.Border[i]
	}
	b.pb[0] = b.padding[style.Left] + b.padding[style.Right] + b.border[style.Left] + b.border[style.Right]
	b.pb[1] = b.padding[style.Top] + b.padding[style.Bottom] + b.border[style.Top] + b.border[style.Bottom]
	sizes := [2]style.Length{st.Width, st.Height}
	mins := [2]style.Length{st.MinWidth, st.MinHeight}
	maxs := [2]style.Length{st.MaxWidth, st.MaxHeight}
	for a := 0; a < 2; a++ {
		extra := geom.Unit(0)
		if st.BoxSizing == style.ContentBox {
			extra = b.pb[a]
		}
		if v, ok := sizes[a].Resolve(cb[a].V, cb[a].OK); ok {
			b.size[a] = some(geom.Max(v+extra, b.pb[a]))
		}
		if mins[a].IsAuto() {
			b.minAuto[a] = true
		} else if v, ok := mins[a].Resolve(cb[a].V, cb[a].OK); ok {
			b.min[a] = v + extra
		}
		b.min[a] = geom.Max(b.min[a], b.pb[a])
		if v, ok := maxs[a].Resolve(cb[a].V, cb[a].OK); ok {
			b.max[a] = some(geom.Max(v+extra, b.pb[a]))
		}
	}
	return b
}

// clamp applies min/max for axis a (min wins over max, as in CSS).
func (b *boxStyle) clamp(a int, v geom.Unit) geom.Unit {
	if b.max[a].OK && v > b.max[a].V {
		v = b.max[a].V
	}
	if v < b.min[a] {
		v = b.min[a]
	}
	return v
}

// clampIf clamps a content-based size, except on an axis whose content
// size is being measured (in.ignoreSize), where min/max must not apply
// (the flex base size ignores them, CSS Flexbox §9.2.3).
func (b *boxStyle) clampIf(a int, v geom.Unit, in input) geom.Unit {
	if in.ignoreSize[a] {
		return geom.Max(v, b.pb[a])
	}
	return b.clamp(a, v)
}

// axisSides returns the physical start and end side indices for axis a.
func axisSides(a int) (start, end int) {
	if a == 0 {
		return style.Left, style.Right
	}
	return style.Top, style.Bottom
}

func edges(s [4]geom.Unit) geom.Edges {
	return geom.Edges{Top: s[style.Top], Right: s[style.Right], Bottom: s[style.Bottom], Left: s[style.Left]}
}

// relativeOffset returns the position: relative shift of an element.
func relativeOffset(st *style.Style, cb [2]Opt) (dx, dy geom.Unit) {
	if st.Position != style.PositionRelative {
		return 0, 0
	}
	if v, ok := st.Inset[style.Left].Resolve(cb[0].V, cb[0].OK); ok {
		dx = v
	} else if v, ok := st.Inset[style.Right].Resolve(cb[0].V, cb[0].OK); ok {
		dx = -v
	}
	if v, ok := st.Inset[style.Top].Resolve(cb[1].V, cb[1].OK); ok {
		dy = v
	} else if v, ok := st.Inset[style.Bottom].Resolve(cb[1].V, cb[1].OK); ok {
		dy = -v
	}
	return dx, dy
}
