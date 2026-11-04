package render

import (
	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Restyle recomputes in place the styles of the given elements (whose
// pseudo-class state changed) and of everything inside them. If only
// paint properties change (style.SameLayout), the boxes keep their layout
// and Restyle returns true; then the caller only needs to repaint.
// Otherwise nothing is modified and it returns false: rebuild with
// Cache.Style and lay out again.
func (d *Document) Restyle(changed []*dom.Node) bool {
	isChanged := map[*dom.Node]bool{}
	for _, n := range changed {
		isChanged[n] = true
	}
	type update struct {
		b  *Box
		st style.Style
	}
	var updates []update
	var visit func(n *dom.Node, parent *style.Style) bool
	visit = func(n *dom.Node, parent *style.Style) bool {
		b := d.boxes[n]
		if b == nil {
			return false // an element without a box (dropped inline group)
		}
		st, _ := d.cascade.Compute(n, parent, n.Attr("style"))
		cmp := st
		if !b.Inline && cmp.Display == style.DisplayInline {
			cmp.Display = style.DisplayFlex // blockified when built
		}
		if b.Inline != (st.Display == style.DisplayInline) || !style.SameLayout(&b.Style, &cmp) {
			return false
		}
		updates = append(updates, update{b, cmp})
		for _, k := range b.Kids {
			if k.Text == nil {
				continue
			}
			// The anonymous box inherits from the new style.
			as := style.Inherit(&cmp)
			if cmp.TextOverflow == style.TextOverflowEllipsis && cmp.Overflow != style.OverflowVisible {
				as.Overflow, as.TextOverflow = cmp.Overflow, cmp.TextOverflow
			}
			as.Finish()
			if !style.SameLayout(&k.Style, &as) {
				return false
			}
			updates = append(updates, update{k, as})
		}
		for _, c := range n.Children {
			if c.Kind == dom.ElementNode && !visit(c, &cmp) {
				return false
			}
		}
		return true
	}
	for _, n := range changed {
		root := true
		for p := n.Parent; p != nil; p = p.Parent {
			if isChanged[p] {
				root = false
				break
			}
		}
		if !root {
			continue
		}
		var parent *style.Style
		if n.Parent != nil {
			pb := d.boxes[n.Parent]
			if pb == nil {
				return false
			}
			parent = &pb.Style
		}
		if !visit(n, parent) {
			return false
		}
	}
	for _, u := range updates {
		u.b.Style = u.st
	}
	return true
}
