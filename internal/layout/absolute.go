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
	cbx := b.border[style.Left]
	cby := b.border[style.Top]
	kb := resolveBox(cs, cbOpt) // auto margins resolve to 0

	var start, end [2]Opt
	for a := 0; a < 2; a++ {
		s, e := axisSides(a)
		if v, ok := cs.Inset[s].Resolve(cb[a], true); ok {
			start[a] = some(v)
		}
		if v, ok := cs.Inset[e].Resolve(cb[a], true); ok {
			end[a] = some(v)
		}
	}
	marg := func(a int) (geom.Unit, geom.Unit) {
		s, e := axisSides(a)
		return kb.margin[s], kb.margin[e]
	}

	var sz [2]Opt
	var av [2]Avail
	var def [2]bool
	for a := 0; a < 2; a++ {
		ms, me := marg(a)
		if kb.size[a].OK {
			sz[a] = some(kb.clamp(a, kb.size[a].V))
		} else if start[a].OK && end[a].OK {
			sz[a] = some(kb.clamp(a, cb[a]-start[a].V-end[a].V-ms-me))
		}
		if sz[a].OK {
			av[a] = Avail{Definite, sz[a].V}
			def[a] = true
		} else {
			room := cb[a] - ms - me
			if start[a].OK {
				room -= start[a].V
			}
			if end[a].OK {
				room -= end[a].V
			}
			av[a] = Avail{Definite, geom.Max(0, room)}
		}
	}
	// Width first (it may determine the height), then height.
	if !sz[0].OK {
		r := ch.compute(input{known: sz, definite: def, parent: cbOpt, avail: av, wantW: true})
		sz[0] = some(r.W)
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
		ms, me := marg(a)
		switch {
		case start[a].OK:
			pos[a] = start[a].V + ms
		case end[a].OK:
			pos[a] = cb[a] - end[a].V - sz[a].V - me
		default:
			// Static position: as if it were the only flex item.
			var mode distMode
			if a == m {
				mode = justifyMode(st.JustifyContent)
				switch mode {
				case distBetween:
					mode = distStart
				case distAround, distEvenly:
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
			free := innerSize[a] - sz[a].V - ms - me
			pos[a] = padStart[a] + ms
			switch mode {
			case distEnd:
				pos[a] += free
			case distCenter:
				pos[a] += part(free, 1, 2)
			}
		}
	}
	ch.Box = Box{
		X: cbx + pos[0], Y: cby + pos[1],
		W: sz[0].V, H: sz[1].V,
		Margin: edges(kb.margin),
	}
	ch.compute(input{
		known:    sz,
		definite: def,
		parent:   cbOpt,
		avail:    [2]Avail{{Definite, sz[0].V}, {Definite, sz[1].V}},
		perform:  true,
	})
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
