package layout

import (
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

type flexItem struct {
	n *Node
	b boxStyle

	align  style.Align // resolved align-self, never AlignAuto
	grow   int64
	shrink int64

	// Physical side indices of main-start, main-end, cross-start,
	// cross-end margins, and the used margin values.
	sMS, sME, sCS, sCE int
	mS, mE, cS, cE     geom.Unit
	aMS, aME, aCS, aCE bool

	base          geom.Unit // flex base size (border box)
	hypo          geom.Unit // hypothetical main size
	minMain       geom.Unit // used min main size (incl. automatic minimum)
	minContrib    geom.Unit // min-content contribution (border box)
	maxContrib    geom.Unit // max-content contribution (border box)
	basisDefinite bool      // flex-basis is a definite length
	target        geom.Unit // resolved main size
	frozen        bool
	viol          geom.Unit

	mainDefinite  bool // post-flex main size is definite (§9.8)
	crossDefinite bool // used cross size is definite
	hypoCross     geom.Unit
	cross         geom.Unit
	mainPos       geom.Unit // border-box start in forward main coordinates
	crossPos      geom.Unit // border-box start in forward cross coordinates
}

func (it *flexItem) outerHypo() geom.Unit   { return it.hypo + it.mS + it.mE }
func (it *flexItem) outerTarget() geom.Unit { return it.target + it.mS + it.mE }

// stretches reports whether align-self: stretch applies: only when the
// cross size property is literally auto (an unresolvable percentage is not
// auto) and neither cross margin is auto.
func (it *flexItem) stretches(c int) bool {
	cs := it.n.Style
	size := cs.Width
	if c == 1 {
		size = cs.Height
	}
	return it.align == style.AlignStretch && size.IsAuto() && !it.aCS && !it.aCE
}

func (it *flexItem) inflexible(c geom.Unit) geom.Unit {
	if (c > it.hypo && it.grow == 0) || (c < it.hypo && it.shrink == 0) {
		return it.hypo
	}
	return c
}

func (it *flexItem) clampMain(m int, v geom.Unit) geom.Unit {
	if it.b.max[m].OK && v > it.b.max[m].V {
		v = it.b.max[m].V
	}
	return geom.Max(v, it.minMain)
}

type flexLine struct {
	start, end int // item index range
	cross      geom.Unit
	offset     geom.Unit // cross start of the line in forward coordinates
}

const infinite = geom.Unit(1) << 50

// flex lays out (or measures) a flex container. See CSS Flexbox §9.
func (n *Node) flex(in input) geom.Size {
	st := n.Style
	b := resolveBox(st, in.parent)
	row := st.FlexDirection.IsRow()
	m, c := 1, 0 // main and cross axis indices (0 = horizontal)
	if row {
		m, c = 0, 1
	}
	wrap := st.FlexWrap != style.NoWrap
	// Browsers size a column-wrap container's min-content width as if it
	// did not wrap (the widest item); line breaking is skipped for that
	// measurement.
	breakLines := wrap && !(!row && in.avail[0].Kind == MinContent)

	// §9.2: available space and the container's own definite sizes.
	// inner: content-box size once known. pct: content-box size if it is
	// definite, used for the percentages of children.
	size, definite := ownSizes(&b, in)
	if in.wantW && size[0].OK {
		return geom.Size{W: size[0].V}
	}
	// Width first: heights depend on widths, and horizontal percentages
	// resolve against the final width. So when the width is not known,
	// determine it with a (cheap, cached) width-only query, then lay out
	// at that width.
	if !size[0].OK && !in.ignoreSize[0] {
		switch {
		case !row && in.avail[0].Kind == Definite:
			// Column container: fit-content,
			// max(min-content, min(max-content, available)).
			probe := in
			probe.perform, probe.wantW = false, true
			probe.avail[0] = Avail{Kind: MaxContent}
			maxc := n.compute(probe).W
			probe.avail[0] = Avail{Kind: MinContent}
			minc := n.compute(probe).W
			size[0] = some(b.clamp(0, geom.Max(minc, geom.Min(maxc, in.avail[0].V))))
			definite[0] = true
			if in.wantW {
				return geom.Size{W: size[0].V}
			}
		case row && !in.wantW:
			probe := in
			probe.perform, probe.wantW = false, true
			size[0] = some(n.compute(probe).W)
			definite[0] = true
		}
	}
	var inner, pct [2]Opt
	var avail [2]Avail
	for a := 0; a < 2; a++ {
		if size[a].OK {
			inner[a] = some(geom.Max(0, size[a].V-b.pb[a]))
			avail[a] = Avail{Definite, inner[a].V}
			if definite[a] {
				pct[a] = inner[a]
			}
		} else {
			avail[a] = innerAvail(in.avail[a], b.pb[a])
		}
	}
	if in.wantW && !row && !wrap {
		return geom.Size{W: n.columnWidth(&b, in, pct, avail)}
	}
	gapCol, _ := st.ColumnGap.Resolve(pct[0].V, pct[0].OK)
	gapRow, _ := st.RowGap.Resolve(pct[1].V, pct[1].OK)
	gapMain, gapCross := gapRow, gapCol
	if row {
		gapMain, gapCross = gapCol, gapRow
	}

	// §9.1-9.2: collect items and determine base and hypothetical sizes.
	items := make([]flexItem, 0, len(n.Children))
	var abs []*Node
	for _, ch := range n.Children {
		switch {
		case ch.Style.Display == style.DisplayNone:
			zeroBoxes(ch)
		case ch.Style.Position == style.PositionAbsolute:
			abs = append(abs, ch)
		default:
			items = append(items, n.newItem(ch, m, c, wrap, inner, pct, avail))
		}
	}

	// §9.2.4: container main size, if it is not definite.
	if !inner[m].OK {
		var v geom.Unit
		if row {
			// Inline axis: fit-content from the items' min-/max-content
			// contributions (as browsers do; flex base sizes are not used).
			maxc := maxContentMain(items, gapMain)
			minc := minContentMain(items, gapMain, wrap)
			switch avail[m].Kind {
			case MaxContent:
				v = maxc
			case MinContent:
				v = minc
			default:
				v = geom.Max(minc, geom.Min(maxc, avail[m].V))
			}
		} else {
			// Block axis: an auto height is the content height.
			v = longestLine(items, gapMain, wrap, colLimit(&b, size, definite, m))
		}
		v = b.clampIf(m, v+b.pb[m], in) - b.pb[m]
		inner[m] = some(v)
		size[m] = some(v + b.pb[m])
	}
	if in.wantW && row {
		return geom.Size{W: size[0].V}
	}
	innerMain := inner[m].V

	// §9.3: collect items into lines.
	limit := innerMain
	if !row {
		limit = colLimit(&b, size, definite, m)
	}
	lines := collectLines(items, limit, gapMain, breakLines)

	// §9.7: resolve flexible lengths.
	for _, ln := range lines {
		resolveFlexible(items[ln.start:ln.end], innerMain, gapMain, m)
	}

	// §9.4: hypothetical cross sizes.
	for i := range items {
		it := &items[i]
		if it.b.size[c].OK {
			it.hypoCross = it.b.clamp(c, it.b.size[c].V)
			continue
		}
		var kn [2]Opt
		var av [2]Avail
		var def [2]bool
		kn[m], def[m] = some(it.target), it.mainDefinite
		av[m] = Avail{Definite, it.target}
		av[c] = shrinkBy(avail[c], it.cS+it.cE)
		sz := it.n.compute(input{known: kn, definite: def, parent: pct, avail: av, wantW: c == 0})
		it.hypoCross = it.b.clamp(c, pick(sz, c))
	}

	// §9.4.8: line cross sizes.
	for li := range lines {
		ln := &lines[li]
		for i := ln.start; i < ln.end; i++ {
			it := &items[i]
			ln.cross = geom.Max(ln.cross, it.hypoCross+it.cS+it.cE)
		}
	}
	if wrap && !row {
		// Browsers re-fit non-stretched items of a column-wrap container
		// to their line: fit-content within the line's cross size.
		for _, ln := range lines {
			for i := ln.start; i < ln.end; i++ {
				it := &items[i]
				if it.b.size[c].OK || it.stretches(c) {
					continue
				}
				var av [2]Avail
				av[0] = Avail{Definite, geom.Max(0, ln.cross-it.cS-it.cE)}
				av[1] = Avail{Kind: MaxContent}
				w := it.n.compute(input{parent: pct, avail: av, wantW: true}).W
				it.hypoCross = it.b.clamp(c, w)
			}
		}
	}
	if !wrap && inner[c].OK && len(lines) == 1 {
		lines[0].cross = inner[c].V
	}
	if !inner[c].OK {
		total := gapCross * geom.Unit(max(len(lines)-1, 0))
		for _, ln := range lines {
			total += ln.cross
		}
		v := b.clampIf(c, total+b.pb[c], in) - b.pb[c]
		inner[c] = some(v)
		size[c] = some(v + b.pb[c])
		if !wrap && len(lines) == 1 {
			lines[0].cross = v
		}
	}
	innerCross := inner[c].V

	// §9.6.15 align-content (multi-line containers only).
	if wrap && len(lines) > 0 {
		free := innerCross - gapCross*geom.Unit(len(lines)-1)
		for _, ln := range lines {
			free -= ln.cross
		}
		mode := distStart
		if st.AlignContent == style.ContentStretch {
			if free > 0 {
				step := free / geom.Unit(len(lines))
				for li := range lines {
					lines[li].cross += step
				}
			}
		} else {
			mode = contentMode(st.AlignContent)
		}
		pos := geom.Unit(0)
		for li := range lines {
			lines[li].offset = pos + spaceBefore(mode, free, li, len(lines), st.FlexWrap == style.WrapReverse)
			pos += lines[li].cross + gapCross
		}
	}

	// §9.4.11 stretch and §9.6 cross-axis alignment within each line.
	for _, ln := range lines {
		for i := ln.start; i < ln.end; i++ {
			it := &items[i]
			it.cross = it.hypoCross
			it.crossDefinite = it.b.size[c].OK
			if it.stretches(c) {
				it.cross = it.b.clamp(c, ln.cross-it.cS-it.cE)
				// §9.8 rule 3: once line cross sizes are known, stretched
				// items count as definite, even in auto-sized containers.
				it.crossDefinite = true
			}
			free := ln.cross - it.cross - it.cS - it.cE
			var off geom.Unit
			switch {
			case it.aCS || it.aCE:
				if free > 0 {
					switch {
					case it.aCS && it.aCE:
						it.cS += part(free, 1, 2)
						it.cE += free - part(free, 1, 2)
					case it.aCS:
						it.cS += free
					default:
						it.cE += free
					}
				}
			case it.align == style.AlignFlexEnd:
				off = free
			case it.align == style.AlignCenter:
				off = part(free, 1, 2)
			}
			it.crossPos = ln.offset + off + it.cS
		}
	}

	// §9.5 main-axis alignment.
	for _, ln := range lines {
		justifyLine(items[ln.start:ln.end], innerMain, gapMain, st.JustifyContent, st.FlexDirection.IsReverse())
	}

	out := geom.Size{W: size[0].V, H: size[1].V}
	if !in.perform {
		return out
	}

	// Final placement.
	n.Box.Border = edges(b.border)
	n.Box.Padding = edges(b.padding)
	origin := [2]geom.Unit{
		b.border[style.Left] + b.padding[style.Left],
		b.border[style.Top] + b.padding[style.Top],
	}
	reverse := st.FlexDirection.IsReverse()
	wrapRev := st.FlexWrap == style.WrapReverse
	for i := range items {
		it := &items[i]
		var pos, sz [2]geom.Unit
		pos[m], sz[m] = it.mainPos, it.target
		pos[c], sz[c] = it.crossPos, it.cross
		if reverse {
			pos[m] = innerMain - pos[m] - sz[m]
		}
		if wrapRev {
			pos[c] = innerCross - pos[c] - sz[c]
		}
		var margin [4]geom.Unit
		margin[it.sMS], margin[it.sME] = it.mS, it.mE
		margin[it.sCS], margin[it.sCE] = it.cS, it.cE
		dx, dy := relativeOffset(it.n.Style, pct)
		var def [2]bool
		def[m], def[c] = it.mainDefinite, it.crossDefinite
		it.n.Box = Box{
			X: origin[0] + pos[0] + dx, Y: origin[1] + pos[1] + dy,
			W: sz[0], H: sz[1],
			Margin: edges(margin),
		}
		it.n.compute(input{
			known:    [2]Opt{some(sz[0]), some(sz[1])},
			definite: def,
			parent:   pct,
			avail:    [2]Avail{{Definite, sz[0]}, {Definite, sz[1]}},
			perform:  true,
		})
	}
	for _, ch := range abs {
		n.layoutAbsolute(ch, &b, [2]geom.Unit{out.W, out.H}, m)
	}
	n.contentExtent(&b, out)
	return out
}

// colLimit is where a column-wrap container breaks lines: its definite
// height, else its specified height (even while its content size is being
// measured) or max-height, else nowhere. A height that merely came from
// content does not break lines (as in browsers).
func colLimit(b *boxStyle, size [2]Opt, definite [2]bool, m int) geom.Unit {
	switch {
	case size[m].OK && definite[m]:
		return geom.Max(0, size[m].V-b.pb[m])
	case b.size[m].OK:
		return geom.Max(0, b.clamp(m, b.size[m].V)-b.pb[m])
	case b.max[m].OK:
		return geom.Max(0, b.max[m].V-b.pb[m])
	}
	return infinite
}

// columnWidth is the width of a single-line column container whose width
// is not known: the widest outer item width (the line's cross size). It
// is equivalent to the full algorithm but skips all height computations.
func (n *Node) columnWidth(b *boxStyle, in input, pct [2]Opt, avail [2]Avail) geom.Unit {
	var w geom.Unit
	for _, ch := range n.Children {
		if ch.Style.Display == style.DisplayNone || ch.Style.Position == style.PositionAbsolute {
			continue
		}
		kb := resolveBox(ch.Style, pct)
		ms := kb.margin[style.Left] + kb.margin[style.Right]
		var cw geom.Unit
		if kb.size[0].OK {
			cw = kb.clamp(0, kb.size[0].V)
		} else {
			cw = ch.compute(input{
				wantW:  true,
				parent: pct,
				avail:  [2]Avail{shrinkBy(avail[0], ms), {Kind: MaxContent}},
			}).W
		}
		w = geom.Max(w, cw+ms)
	}
	return b.clampIf(0, w+b.pb[0], in)
}

// newItem resolves a child's flex-item properties and sizes (§9.2.3).
func (n *Node) newItem(ch *Node, m, c int, wrap bool, inner, pct [2]Opt, avail [2]Avail) flexItem {
	st := n.Style
	cs := ch.Style
	it := flexItem{n: ch, b: resolveBox(cs, pct)}
	// §9.8: the flexed main size is definite if the container's main size
	// is, or the item has a definite size or a definite flex basis of its
	// own (the latter as browsers do).
	_, basisDef := cs.FlexBasis.Resolve(pct[m].V, pct[m].OK)
	it.basisDefinite = basisDef
	it.mainDefinite = pct[m].OK || basisDef || (cs.FlexBasis.IsAuto() && it.b.size[m].OK)
	it.align = cs.AlignSelf
	if it.align == style.AlignAuto {
		it.align = st.AlignItems
	}
	it.grow, it.shrink = int64(cs.FlexGrow), int64(cs.FlexShrink)

	it.sMS, it.sME = axisSides(m)
	it.sCS, it.sCE = axisSides(c)
	if st.FlexDirection.IsReverse() {
		it.sMS, it.sME = it.sME, it.sMS
	}
	if st.FlexWrap == style.WrapReverse {
		it.sCS, it.sCE = it.sCE, it.sCS
	}
	it.mS, it.mE = it.b.margin[it.sMS], it.b.margin[it.sME]
	it.cS, it.cE = it.b.margin[it.sCS], it.b.margin[it.sCE]
	it.aMS, it.aME = it.b.mAuto[it.sMS], it.b.mAuto[it.sME]
	it.aCS, it.aCE = it.b.mAuto[it.sCS], it.b.mAuto[it.sCE]

	// Cross size used while measuring the main size: the definite cross
	// size, or the stretched size in a single-line container, or else
	// fit-content within the available cross space.
	crossKnown := Opt{}
	crossDef := false
	if it.b.size[c].OK {
		crossKnown, crossDef = some(it.b.clamp(c, it.b.size[c].V)), true
	} else if it.stretches(c) && !wrap && inner[c].OK {
		crossKnown, crossDef = some(it.b.clamp(c, inner[c].V-it.cS-it.cE)), pct[c].OK
	}
	crossAvail := shrinkBy(avail[c], it.cS+it.cE)
	if crossKnown.OK {
		crossAvail = Avail{Definite, crossKnown.V}
	}
	// measure returns the content-based main size (the item's own main
	// size property is ignored, as for flex-basis: content).
	measure := func(kind AvailKind) geom.Unit {
		var in input
		in.known[c], in.definite[c] = crossKnown, crossDef
		in.ignoreSize[m] = true
		in.avail[m] = Avail{Kind: kind}
		in.avail[c] = crossAvail
		in.parent = pct
		in.wantW = m == 0
		return pick(ch.compute(in), m)
	}

	// Flex base size.
	maxContent := geom.Unit(-1)
	pbMain := it.b.pb[m]
	if v, ok := cs.FlexBasis.Resolve(pct[m].V, pct[m].OK); ok {
		if cs.BoxSizing == style.ContentBox {
			v += pbMain
		}
		it.base = v
	} else if cs.FlexBasis.IsAuto() && it.b.size[m].OK {
		// flex-basis: auto uses the main size property.
		it.base = it.b.size[m].V
	} else {
		maxContent = measure(MaxContent)
		it.base = maxContent
	}
	it.base = geom.Max(it.base, pbMain)

	// Min-content size and automatic minimum size (§4.5).
	visible := cs.Overflow == style.OverflowVisible
	needMin := !inner[m].OK || (it.b.minAuto[m] && visible)
	var minContent geom.Unit
	if needMin {
		minContent = measure(MinContent)
	}
	it.minMain = it.b.min[m]
	if it.b.minAuto[m] && visible {
		auto := minContent
		if it.b.max[m].OK {
			auto = geom.Min(auto, it.b.max[m].V)
		}
		if it.b.size[m].OK {
			auto = geom.Min(auto, it.b.size[m].V)
		}
		it.minMain = geom.Max(it.minMain, auto)
	}
	it.hypo = it.clampMain(m, it.base)
	// Contributions: the preferred size if definite, else the min-/max-
	// content size, clamped by min/max. Only needed to size the container.
	if !inner[m].OK {
		if it.b.size[m].OK {
			it.minContrib = it.clampMain(m, it.b.size[m].V)
			it.maxContrib = it.minContrib
		} else {
			it.minContrib = it.clampMain(m, minContent)
			if maxContent < 0 {
				maxContent = measure(MaxContent)
			}
			it.maxContrib = it.clampMain(m, maxContent)
		}
		// With an explicit flex-basis, an item that cannot grow does not
		// contribute more than its hypothetical main size, and one that
		// cannot shrink not less (matches Chromium; not CSS §9.9.1).
		if it.basisDefinite && !wrap {
			it.minContrib = it.inflexible(it.minContrib)
			it.maxContrib = it.inflexible(it.maxContrib)
		}
	}
	return it
}

func pick(s geom.Size, a int) geom.Unit {
	if a == 0 {
		return s.W
	}
	return s.H
}

func shrinkBy(a Avail, by geom.Unit) Avail {
	if a.Kind == Definite {
		a.V = geom.Max(0, a.V-by)
	}
	return a
}

// longestLine returns the largest sum of outer hypothetical main sizes
// (plus gaps) of any line when breaking at limit.
func longestLine(items []flexItem, gap geom.Unit, wrap bool, limit geom.Unit) geom.Unit {
	var best, cur geom.Unit
	count := 0
	for i := range items {
		o := items[i].outerHypo()
		if count > 0 && wrap && cur+gap+o > limit {
			best = geom.Max(best, cur)
			cur, count = 0, 0
		}
		if count > 0 {
			cur += gap
		}
		cur += o
		count++
	}
	return geom.Max(best, cur)
}

// maxContentMain returns the max-content inner main size of a row
// container with all items on one line: the sum of the items' max-content
// contributions (see newItem for how flex-basis adjusts them).
func maxContentMain(items []flexItem, gap geom.Unit) geom.Unit {
	var v geom.Unit
	for i := range items {
		if i > 0 {
			v += gap
		}
		v += items[i].maxContrib + items[i].mS + items[i].mE
	}
	return v
}

// minContentMain returns the container's min-content inner main size: the
// sum of min-content contributions (largest one if the container wraps).
func minContentMain(items []flexItem, gap geom.Unit, wrap bool) geom.Unit {
	var v geom.Unit
	for i := range items {
		o := items[i].minContrib + items[i].mS + items[i].mE
		if wrap {
			v = geom.Max(v, o)
		} else {
			if i > 0 {
				v += gap
			}
			v += o
		}
	}
	return v
}

func collectLines(items []flexItem, innerMain, gap geom.Unit, wrap bool) []flexLine {
	if len(items) == 0 {
		return []flexLine{{}}
	}
	if !wrap {
		return []flexLine{{start: 0, end: len(items)}}
	}
	var lines []flexLine
	start := 0
	var cur geom.Unit
	for i := range items {
		o := items[i].outerHypo()
		if i > start && cur+gap+o > innerMain {
			lines = append(lines, flexLine{start: start, end: i})
			start, cur = i, 0
		}
		if i > start {
			cur += gap
		}
		cur += o
	}
	return append(lines, flexLine{start: start, end: len(items)})
}

// resolveFlexible implements §9.7 for one line.
func resolveFlexible(items []flexItem, innerMain, gap geom.Unit, m int) {
	if len(items) == 0 {
		return
	}
	gaps := gap * geom.Unit(len(items)-1)
	used := gaps
	for i := range items {
		used += items[i].outerHypo()
	}
	growing := used < innerMain
	factor := func(it *flexItem) int64 {
		if growing {
			return it.grow
		}
		return it.shrink
	}
	for i := range items {
		it := &items[i]
		it.target, it.frozen = it.base, false
		if factor(it) == 0 || (growing && it.base > it.hypo) || (!growing && it.base < it.hypo) {
			it.target, it.frozen = it.hypo, true
		}
	}
	remaining := func() geom.Unit {
		free := innerMain - gaps
		for i := range items {
			it := &items[i]
			free -= it.mS + it.mE
			if it.frozen {
				free -= it.target
			} else {
				free -= it.base
			}
		}
		return free
	}
	initialFree := remaining()
	for iter := 0; iter <= len(items); iter++ {
		var sumFactors int64
		unfrozen := 0
		for i := range items {
			if !items[i].frozen {
				sumFactors += factor(&items[i])
				unfrozen++
			}
		}
		if unfrozen == 0 {
			break
		}
		free := remaining()
		if sumFactors < style.FactorScale {
			cand := geom.Unit(geom.MulDiv(int64(initialFree), sumFactors, style.FactorScale))
			if abs(cand) < abs(free) {
				free = cand
			}
		}
		// Distribute free space in proportion to the (scaled) factors.
		weight := func(it *flexItem) int64 {
			if growing {
				return it.grow
			}
			return it.shrink * int64(it.base-it.b.pb[m])
		}
		var sumW int64
		for i := range items {
			if !items[i].frozen {
				sumW += weight(&items[i])
			}
		}
		var cum int64
		var prev geom.Unit
		for i := range items {
			it := &items[i]
			if it.frozen {
				continue
			}
			it.target = it.base
			if free != 0 && sumW > 0 {
				cum += weight(it)
				now := geom.Unit(geom.MulDiv(int64(free), cum, sumW))
				it.target += now - prev
				prev = now
			}
		}
		// Fix min/max violations and freeze.
		var total geom.Unit
		for i := range items {
			it := &items[i]
			if it.frozen {
				continue
			}
			cl := it.clampMain(m, it.target)
			it.viol = cl - it.target
			it.target = cl
			total += it.viol
		}
		for i := range items {
			it := &items[i]
			if it.frozen {
				continue
			}
			if total == 0 || (total > 0 && it.viol > 0) || (total < 0 && it.viol < 0) {
				it.frozen = true
			}
		}
	}
}

func abs(v geom.Unit) geom.Unit {
	if v < 0 {
		return -v
	}
	return v
}

// justifyLine resolves main-axis auto margins and justify-content (§9.5),
// setting mainPos (forward coordinates) for each item.
func justifyLine(items []flexItem, innerMain, gap geom.Unit, j style.Justify, reverse bool) {
	if len(items) == 0 {
		return
	}
	free := innerMain - gap*geom.Unit(len(items)-1)
	autos := 0
	for i := range items {
		free -= items[i].outerTarget()
		if items[i].aMS {
			autos++
		}
		if items[i].aME {
			autos++
		}
	}
	if free > 0 && autos > 0 {
		k := 0
		var prev geom.Unit
		next := func() geom.Unit {
			k++
			now := part(free, int64(k), int64(autos))
			d := now - prev
			prev = now
			return d
		}
		for i := range items {
			if items[i].aMS {
				items[i].mS += next()
			}
			if items[i].aME {
				items[i].mE += next()
			}
		}
		free = 0
	}
	mode := justifyMode(j)
	if !reverse {
		pos := geom.Unit(0)
		for i := range items {
			it := &items[i]
			it.mainPos = pos + spaceBefore(mode, free, i, len(items), false) + it.mS
			pos += it.outerTarget() + gap
		}
		return
	}
	// Reversed: browsers place items in physical order (last item first),
	// so truncation remainders end up at the physical end. Compute
	// physical positions, then convert back to forward coordinates (the
	// caller mirrors them).
	switch {
	case mode == distBetween && (free < 0 || len(items) < 2):
		mode = distEnd // flex-start = physical end
	case free < 0 && (mode == distAround || mode == distEvenly):
		mode = distStart // safe fallback: writing-mode start
	default:
		mode = flipMode(mode)
	}
	pos := geom.Unit(0)
	n := len(items)
	for j := 0; j < n; j++ {
		it := &items[n-1-j]
		phys := pos + spaceBefore(mode, free, j, n, false) + it.mE
		it.mainPos = innerMain - phys - it.target
		pos += it.outerTarget() + gap
	}
}

type distMode uint8

const (
	distStart distMode = iota
	distEnd
	distCenter
	distBetween
	distAround
	distEvenly
)

func justifyMode(j style.Justify) distMode {
	switch j {
	case style.JustifyFlexEnd:
		return distEnd
	case style.JustifyCenter:
		return distCenter
	case style.JustifySpaceBetween:
		return distBetween
	case style.JustifySpaceAround:
		return distAround
	case style.JustifySpaceEvenly:
		return distEvenly
	}
	return distStart
}

func contentMode(a style.AlignContent) distMode {
	switch a {
	case style.ContentFlexEnd:
		return distEnd
	case style.ContentCenter:
		return distCenter
	case style.ContentSpaceBetween:
		return distBetween
	case style.ContentSpaceAround:
		return distAround
	case style.ContentSpaceEvenly:
		return distEvenly
	}
	return distStart
}

// spaceBefore returns the total extra space placed before item i of n
// when distributing free space with the given mode. Steps are truncated
// per gap, as browsers do (the remainder stays at the end). reverse says
// that main-start is the physical end (row-reverse/column-reverse, or
// wrap-reverse for lines).
func spaceBefore(mode distMode, free geom.Unit, i, n int, reverse bool) geom.Unit {
	if free < 0 {
		// Overflow fallbacks (CSS Box Alignment): space-between ->
		// flex-start; space-around/evenly -> safe center, i.e. the
		// writing-mode start, which is flex-end in a reversed container.
		switch mode {
		case distBetween:
			mode = distStart
		case distAround, distEvenly:
			mode = distStart
			if reverse {
				mode = distEnd
			}
		}
	}
	k := geom.Unit(i)
	switch mode {
	case distEnd:
		return free
	case distCenter:
		return free / 2
	case distBetween:
		if n < 2 {
			return 0
		}
		return k * (free / geom.Unit(n-1))
	case distAround:
		step := free / geom.Unit(n)
		return step/2 + k*step
	case distEvenly:
		step := free / geom.Unit(n+1)
		return step + k*step
	}
	return 0
}

// part returns v*num/den truncated toward zero.
func part(v geom.Unit, num, den int64) geom.Unit {
	return geom.Unit(geom.MulDiv(int64(v), num, den))
}

// contentExtent computes the scrollable overflow size of n.
func (n *Node) contentExtent(b *boxStyle, size geom.Size) {
	pw := size.W - b.border[style.Left] - b.border[style.Right]
	ph := size.H - b.border[style.Top] - b.border[style.Bottom]
	cw, ch := pw, ph
	for _, k := range n.Children {
		if k.Style.Display == style.DisplayNone {
			continue
		}
		bx := &k.Box
		ex := bx.X + bx.W + bx.Margin.Right - b.border[style.Left]
		ey := bx.Y + bx.H + bx.Margin.Bottom - b.border[style.Top]
		if k.Style.Position != style.PositionAbsolute {
			ex += b.padding[style.Right]
			ey += b.padding[style.Bottom]
		}
		cw, ch = geom.Max(cw, ex), geom.Max(ch, ey)
	}
	n.Box.ContentW, n.Box.ContentH = cw, ch
}

// zeroBoxes clears the boxes of a display:none subtree.
func zeroBoxes(n *Node) {
	n.Box = Box{}
	for _, c := range n.Children {
		zeroBoxes(c)
	}
}
