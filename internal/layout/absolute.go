package layout

import (
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// layoutAbsolute positions an absolutely positioned child of n
// (spec/SUBSET.md §2.4). b is n's resolved box, size its border-box size,
// m its main axis index.
func (n *Node) layoutAbsolute(ch *Node, b *boxStyle, size [2]geom.Unit, m int) {
	st := n.Style
	cs := ch.Style
	// Containing block: n's padding box.
	cb := [2]geom.Unit{
		size[0] - b.border[style.Left] - b.border[style.Right],
		size[1] - b.border[style.Top] - b.border[style.Bottom],
	}
	cbOpt := [2]Opt{some(cb[0]), some(cb[1])}
	kb := resolveBox(cs, cbOpt)

	var start, end [2]Opt
	var ms, me [2]geom.Unit // used margins (auto = 0 until resolved)
	var autoS, autoE [2]bool
	for a := 0; a < 2; a++ {
		s, e := axisSides(a)
		if v, ok := cs.Inset[s].Resolve(cb[a], true); ok {
			start[a] = some(v)
		}
		if v, ok := cs.Inset[e].Resolve(cb[a], true); ok {
			end[a] = some(v)
		}
		ms[a], me[a] = kb.margin[s], kb.margin[e]
		autoS[a], autoE[a] = kb.mAuto[s], kb.mAuto[e]
	}
	// Vertical self-alignment (CSS Position 3): with both top and bottom
	// set, align-self flex-start/center/flex-end make an auto height
	// fit-content and align the box inside the inset-modified containing
	// block; auto/stretch stretch it. Horizontally (no justify-self in the
	// subset) the box always stretches.
	vAlign := distStart
	vStretch := true
	switch cs.AlignSelf {
	case style.AlignFlexStart:
		vStretch = false
	case style.AlignCenter:
		vAlign, vStretch = distCenter, false
	case style.AlignFlexEnd:
		vAlign, vStretch = distEnd, false
	}
	both := func(a int) bool { return start[a].OK && end[a].OK }

	var sz [2]Opt
	var av [2]Avail
	var def [2]bool
	for a := 0; a < 2; a++ {
		if kb.size[a].OK {
			sz[a] = some(kb.clamp(a, kb.size[a].V))
		} else if both(a) && (a == 0 || vStretch) {
			sz[a] = some(kb.clamp(a, cb[a]-start[a].V-end[a].V-ms[a]-me[a]))
		}
		if sz[a].OK {
			av[a] = Avail{Definite, sz[a].V}
			def[a] = true
		} else {
			room := cb[a] - ms[a] - me[a]
			if start[a].OK {
				room -= start[a].V
			}
			if end[a].OK {
				room -= end[a].V
			}
			if !start[a].OK && !end[a].OK {
				// Shrink-to-fit from the static position (CSS 2 §10.3.7):
				// the room is measured from the side it is anchored to.
				s, e := axisSides(a)
				switch staticMode(st, cs, a, m) {
				case distStart:
					room -= b.padding[s]
				case distEnd:
					room -= b.padding[e]
				}
			}
			av[a] = Avail{Definite, geom.Max(0, room)}
		}
	}
	// Width first (it may determine the height), then height.
	if !sz[0].OK {
		r := ch.compute(input{known: sz, definite: def, parent: cbOpt, avail: av, wantW: true})
		sz[0], def[0] = some(r.W), true
		av[0] = Avail{Definite, r.W}
	}
	if !sz[1].OK {
		r := ch.compute(input{known: sz, definite: def, parent: cbOpt, avail: av})
		sz[1] = some(r.H)
		av[1] = Avail{Definite, r.H}
	}

	padStart := [2]geom.Unit{b.padding[style.Left], b.padding[style.Top]}
	innerSize := [2]geom.Unit{
		cb[0] - b.padding[style.Left] - b.padding[style.Right],
		cb[1] - b.padding[style.Top] - b.padding[style.Bottom],
	}
	var pos [2]geom.Unit
	for a := 0; a < 2; a++ {
		switch {
		case both(a):
			imcb := cb[a] - start[a].V - end[a].V
			imStart := start[a].V
			if imcb < 0 {
				// Negative inset-modified containing block: zero size at
				// the start inset (CSS Position 3).
				imcb = 0
			}
			free := imcb - sz[a].V - ms[a] - me[a]
			switch {
			case autoS[a] && autoE[a]:
				if a == 0 && free < 0 {
					me[a] += free // over-constrained: the end margin gives way
				} else {
					ms[a] += free / 2
					me[a] += free - free/2
				}
				pos[a] = start[a].V + ms[a]
			case autoS[a]:
				ms[a] += free
				pos[a] = start[a].V + ms[a]
			case autoE[a]:
				me[a] += free
				pos[a] = start[a].V + ms[a]
			case a == 1 && !vStretch:
				off := geom.Unit(0)
				switch vAlign {
				case distCenter:
					off = free / 2
				case distEnd:
					off = free
				}
				pos[a] = imStart + off + ms[a]
			default:
				pos[a] = start[a].V + ms[a]
			}
		case start[a].OK:
			pos[a] = start[a].V + ms[a]
		case end[a].OK:
			pos[a] = cb[a] - end[a].V - sz[a].V - me[a]
		default:
			pos[a] = staticPosition(st, cs, a, m, innerSize[a]-sz[a].V-ms[a]-me[a]) + padStart[a] + ms[a]
		}
	}
	var margin [4]geom.Unit
	margin[style.Left], margin[style.Right] = ms[0], me[0]
	margin[style.Top], margin[style.Bottom] = ms[1], me[1]
	ch.Box = Box{
		X: b.border[style.Left] + pos[0], Y: b.border[style.Top] + pos[1],
		W: sz[0].V, H: sz[1].V,
		Margin: edges(margin),
	}
	ch.compute(input{
		known:    sz,
		definite: def,
		parent:   cbOpt,
		avail:    [2]Avail{{Definite, sz[0].V}, {Definite, sz[1].V}},
		perform:  true,
	})
}

// staticPosition returns the offset of an absolutely positioned child
// along axis a within the parent's content box, as if it were the only
// flex item. free is the content-box size minus the child's outer size.
func staticPosition(st, cs *style.Style, a, m int, free geom.Unit) geom.Unit {
	switch staticMode(st, cs, a, m) {
	case distEnd:
		return free
	case distCenter:
		return free / 2
	}
	return 0
}

// staticMode returns how an absolutely positioned child is aligned along
// axis a at its static position: start, center or end.
func staticMode(st, cs *style.Style, a, m int) distMode {
	var mode distMode
	if a == m {
		switch justifyMode(st.JustifyContent) {
		case distEnd:
			mode = distEnd
		case distCenter, distAround, distEvenly:
			mode = distCenter
		}
		if st.FlexDirection.IsReverse() {
			mode = flipMode(mode)
		}
	} else {
		al := cs.AlignSelf
		if al == style.AlignAuto {
			al = st.AlignItems
		}
		switch al {
		case style.AlignFlexEnd:
			mode = distEnd
		case style.AlignCenter:
			mode = distCenter
		}
		if st.FlexWrap == style.WrapReverse {
			mode = flipMode(mode)
		}
	}
	return mode
}

func flipMode(d distMode) distMode {
	switch d {
	case distStart:
		return distEnd
	case distEnd:
		return distStart
	}
	return d
}
