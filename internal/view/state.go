package view

import (
	"sort"
	"strconv"
	"strings"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/render"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// ---- Element keys (state preserved across rebuilds, §8.2) ----

// key identifies an element across rebuilds of the document: "#id" when
// it has an id, otherwise the path of element-child indices from the
// root ("/0/2/1").
func key(n *dom.Node) string {
	if n.ID() != "" {
		return "#" + n.ID()
	}
	var parts []string
	for ; n.Parent != nil; n = n.Parent {
		i := 0
		for _, c := range n.Parent.Children {
			if c == n {
				break
			}
			if c.Kind == dom.ElementNode {
				i++
			}
		}
		parts = append(parts, strconv.Itoa(i))
	}
	var b strings.Builder
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString("/")
		b.WriteString(parts[i])
	}
	return b.String()
}

// findKey finds the element with key k under root.
func findKey(root *dom.Node, k string) *dom.Node {
	if strings.HasPrefix(k, "#") {
		var find func(n *dom.Node) *dom.Node
		find = func(n *dom.Node) *dom.Node {
			if n.Kind == dom.ElementNode && n.ID() == k[1:] {
				return n
			}
			for _, c := range n.Children {
				if r := find(c); r != nil {
					return r
				}
			}
			return nil
		}
		return find(root)
	}
	n := root
	for _, p := range strings.Split(k, "/")[1:] {
		i, _ := strconv.Atoi(p)
		var next *dom.Node
		for _, c := range n.Children {
			if c.Kind != dom.ElementNode {
				continue
			}
			if i == 0 {
				next = c
				break
			}
			i--
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

// ---- Pseudo-class state (§8.3) ----

// setChain moves flag from the old element (and its ancestors, if
// ancestors) to the new one, and invalidates style if a rule uses it.
func (v *View) setChain(old, now *dom.Node, flag style.State, ancestors bool) {
	if old == now {
		return
	}
	affects := func(n *dom.Node) {
		if v.doc != nil && !v.needStyle && v.doc.Affects(n, flag) {
			v.needStyle = true
		}
	}
	for n := old; n != nil; n = n.Parent {
		if n.Dyn&flag != 0 {
			n.Dyn &^= flag
			affects(n)
		}
		if !ancestors {
			break
		}
	}
	for n := now; n != nil; n = n.Parent {
		if n.Dyn&flag == 0 {
			n.Dyn |= flag
			affects(n)
		}
		if !ancestors {
			break
		}
	}
}

// setHover: :hover applies to the element under the pointer and its
// ancestors.
func (v *View) setHover(n *dom.Node) {
	v.setChain(v.hover, n, style.StateHover, true)
	v.hover = n
}

// setActive: :active applies to the pressed element and its ancestors.
func (v *View) setActive(n *dom.Node) {
	v.setChain(v.active, n, style.StateActive, true)
	v.active = n
}

// setFocus: :focus applies to the focused element only. A change sends
// blur to the old element and focus to the new one (they do not bubble).
func (v *View) setFocus(n *dom.Node) {
	old := v.focus
	v.setChain(v.focus, n, style.StateFocus, false)
	v.focus = n
	if old != n {
		if old != nil {
			if st := v.input(old); st != nil {
				v.changed(old, st) // change fires before blur, as in browsers
				v.needPaint = true
			}
			v.dispatchSelf(&Event{Type: "blur", Target: old})
		}
		if n != nil {
			if st := v.input(n); st != nil {
				st.atFocus = st.buf.String()
				v.needPaint = true
			}
			v.dispatchSelf(&Event{Type: "focus", Target: n})
		}
	}
	v.focusKey = ""
	if n != nil {
		v.focusKey = key(n)
	}
}

// FocusID focuses the element with the given id ("" blurs). It reports
// whether such a focusable element exists.
func (v *View) FocusID(id string) bool {
	d := v.Document()
	if d == nil {
		return false
	}
	if id == "" {
		v.setFocus(nil)
		return true
	}
	n := findKey(d.Dom.Root, "#"+id)
	if n == nil || !v.focusable(n) {
		return false
	}
	v.setFocus(n)
	return true
}

// FocusedID returns the id of the focused element ("" if none or if it
// has no id).
func (v *View) FocusedID() string {
	if v.focus == nil {
		return ""
	}
	return v.focus.ID()
}

// tabIndex returns the element's tabindex and whether it is focusable at
// all: button, a, input and elements with a tabindex attribute.
func tabIndex(n *dom.Node) (int, bool) {
	if s := n.Attr("tabindex"); s != "" {
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			return i, true
		}
	}
	switch n.Tag() {
	case "button", "a", "input":
		return 0, true
	}
	return 0, false
}

// focusable: focusable by tag or tabindex, and rendered (not inside a
// display: none subtree).
func (v *View) focusable(n *dom.Node) bool {
	if n == nil || n.Kind != dom.ElementNode {
		return false
	}
	if _, ok := tabIndex(n); !ok || n.Disabled() {
		return false
	}
	for x := n; x != nil; x = x.Parent {
		if b := v.doc.BoxOf(x); b == nil || b.Style.Display == style.DisplayNone {
			return false
		}
	}
	return true
}

// moveFocus moves focus to the next (or previous) element in sequential
// focus order (HTML): positive tabindex values ascending, then tabindex 0
// and naturally focusable elements, each in tree order; tabindex < 0 is
// skipped. It wraps around.
func (v *View) moveFocus(back bool) {
	type item struct {
		n   *dom.Node
		idx int
	}
	var order []item
	var walk func(n *dom.Node)
	walk = func(n *dom.Node) {
		if n.Kind != dom.ElementNode {
			return
		}
		if i, ok := tabIndex(n); ok && i >= 0 && v.focusable(n) {
			order = append(order, item{n, i})
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(v.doc.Dom.Root)
	if len(order) == 0 {
		return
	}
	sort.SliceStable(order, func(a, b int) bool {
		ia, ib := order[a].idx, order[b].idx
		if (ia > 0) != (ib > 0) {
			return ia > 0
		}
		return ia > 0 && ia < ib
	})
	cur := -1
	for i, it := range order {
		if it.n == v.focus {
			cur = i
		}
	}
	var next int
	switch {
	case cur < 0 && back:
		next = len(order) - 1
	case cur < 0:
		next = 0
	case back:
		next = (cur - 1 + len(order)) % len(order)
	default:
		next = (cur + 1) % len(order)
	}
	v.setFocus(order[next].n)
}

// ---- Scrolling (§8.5) ----

// scrollable: overflow: scroll boxes are user-scrollable.
func scrollable(b *render.Box) bool {
	return !b.Inline && b.Lay != nil && b.Style.Overflow == style.OverflowScroll
}

func clampScroll(b *render.Box, o geom.Point) geom.Point {
	m := b.MaxScroll()
	return geom.Point{X: min(max(o.X, 0), m.X), Y: min(max(o.Y, 0), m.Y)}
}

// scrollerFor returns the nearest scroll container (n or an ancestor)
// that can scroll at all.
func (v *View) scrollerFor(n *dom.Node) *dom.Node {
	for ; n != nil; n = n.Parent {
		if b := v.doc.BoxOf(n); b != nil && scrollable(b) && b.MaxScroll() != (geom.Point{}) {
			return n
		}
	}
	return nil
}

// scrollBy scrolls n by (dx, dy), clamped to its range, and returns the
// part it could not absorb and whether it moved.
func (v *View) scrollBy(n *dom.Node, dx, dy geom.Unit, chain bool) (geom.Unit, geom.Unit, bool) {
	b := v.doc.BoxOf(n)
	if b == nil {
		return dx, dy, false
	}
	k := key(n)
	old := v.scroll[k]
	now := clampScroll(b, geom.Point{X: old.X + dx, Y: old.Y + dy})
	if now == old {
		return dx, dy, false
	}
	v.scroll[k] = now
	v.needPlace = true
	return dx - (now.X - old.X), dy - (now.Y - old.Y), true
}

// ScrollOffset returns the scroll offset of the element with the given
// id.
func (v *View) ScrollOffset(id string) geom.Point { return v.scroll["#"+id] }
