package view

import (
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/dom"
)

// TestKeys: every element's key finds it again, in the same document and
// in a re-parsed copy (the case after a re-render).
func TestKeys(t *testing.T) {
	src := `<div id="a"><span>x</span>text<div><p>1</p><p id="b">2</p><p>3</p></div></div><ul><li>a</li><li>b</li></ul>`
	d1, _ := dom.Parse(src)
	d2, _ := dom.Parse(src)
	var walk func(a, b *dom.Node)
	n := 0
	walk = func(a, b *dom.Node) {
		if a.Kind != dom.ElementNode {
			return
		}
		k := key(a)
		if findKey(d1.Root, k) != a || findKey(d2.Root, k) != b {
			t.Errorf("key %q does not find <%s>", k, a.Tag())
		}
		n++
		for i := range a.Children {
			walk(a.Children[i], b.Children[i])
		}
	}
	walk(d1.Root, d2.Root)
	if n != 10 { // body, div, span, div, 3 p, ul, 2 li
		t.Errorf("visited %d elements", n)
	}
	if key(findKey(d1.Root, "#b")) != "#b" || findKey(d1.Root, "/9") != nil || findKey(d1.Root, "#nope") != nil {
		t.Error("lookups of ids / missing paths")
	}
}
