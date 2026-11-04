package style

import (
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

type fakeEl struct {
	tag, id string
	classes []string
	parent  *fakeEl
	state   State
}

func (e *fakeEl) Tag() string { return e.tag }
func (e *fakeEl) ID() string  { return e.id }
func (e *fakeEl) HasClass(c string) bool {
	for _, x := range e.classes {
		if x == c {
			return true
		}
	}
	return false
}
func (e *fakeEl) ParentElement() Element {
	if e.parent == nil {
		return nil
	}
	return e.parent
}
func (e *fakeEl) State() State { return e.state }

func TestColors(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Color
	}{
		{"#fff", Color{255, 255, 255, 255}},
		{"#0a0B0c", Color{10, 11, 12, 255}},
		{"#f008", Color{255, 0, 0, 136}},
		{"#11223344", Color{0x11, 0x22, 0x33, 0x44}},
		{"rgb(1, 2, 3)", Color{1, 2, 3, 255}},
		{"rgba(1,2,3,0.5)", Color{1, 2, 3, 128}},
		{"rgb(10 20 30 / 25%)", Color{10, 20, 30, 64}},
		{"rgb(100%, 50%, 0%)", Color{255, 128, 0, 255}},
		{"rebeccapurple", Color{102, 51, 153, 255}},
		{"TRANSPARENT", Color{}},
	} {
		s, w := parse(t, "background-color: "+c.in)
		if len(w) != 0 || s.BackgroundColor != c.want {
			t.Errorf("%s: got %v (%v), want %v", c.in, s.BackgroundColor, w, c.want)
		}
	}
	for _, bad := range []string{"#ff", "rgb(1,2)", "hsl(0, 0%, 0%)", "notacolor"} {
		if _, w := parse(t, "background-color: "+bad); len(w) == 0 {
			t.Errorf("%s: expected a warning", bad)
		}
	}
}

func TestPaintProperties(t *testing.T) {
	s, w := parse(t, "border: 2px solid rgb(1, 2, 3); border-radius: 4px 50%; opacity: 0.25; "+
		"box-shadow: 0 2px 6px -1px rgba(0, 0, 0, .5); color: red; font-size: 1.5rem; font-weight: bold; "+
		"line-height: 1.25; text-align: center; white-space: nowrap; text-overflow: ellipsis")
	if len(w) != 0 {
		t.Fatal(w)
	}
	s.Finish()
	if s.Border != [4]geom.Unit{geom.Px(2), geom.Px(2), geom.Px(2), geom.Px(2)} || s.BorderColor != (Color{1, 2, 3, 255}) || s.BorderColorCurrent {
		t.Errorf("border %+v %+v", s.Border, s.BorderColor)
	}
	if s.BorderRadius[0][0] != Px(4) || s.BorderRadius[1][1] != Percent(50) {
		t.Errorf("radius %+v", s.BorderRadius)
	}
	if s.Opacity != 250 {
		t.Errorf("opacity %d", s.Opacity)
	}
	want := BoxShadow{Set: true, DY: geom.Px(2), Blur: geom.Px(6), Spread: -geom.Px(1), Color: Color{0, 0, 0, 128}}
	if s.Shadow != want {
		t.Errorf("shadow %+v", s.Shadow)
	}
	if s.Color != (Color{255, 0, 0, 255}) || s.FontSize != geom.Px(24) || s.FontWeight != 700 ||
		s.LineHeight != (LineHeight{LineHeightNumber, 1250}) || s.TextAlign != TextAlignCenter ||
		s.WhiteSpace != WhiteSpaceNowrap || s.TextOverflow != TextOverflowEllipsis {
		t.Errorf("text props %+v", s)
	}
	// currentcolor defaults.
	s, _ = parse(t, "color: #00f; border-width: 1px; box-shadow: 1px 1px")
	s.Finish()
	if s.BorderColor != (Color{0, 0, 255, 255}) || s.Shadow.Color != (Color{0, 0, 255, 255}) {
		t.Errorf("currentcolor: %+v %+v", s.BorderColor, s.Shadow)
	}
	// border without a style draws nothing (as in CSS).
	if s, _ := parse(t, "border: 3px red"); s.Border[0] != 0 {
		t.Errorf("border without style: %v", s.Border)
	}
	for _, bad := range []string{"box-shadow: inset 0 0 1px red", "box-shadow: 1px 1px red, 2px 2px blue",
		"border-color: red blue", "border-radius: 1px / 2px", "border-style: dashed", "background: url(x.png)"} {
		if _, w := parse(t, bad); len(w) == 0 {
			t.Errorf("%q: expected a warning", bad)
		}
	}
}

func TestSelectors(t *testing.T) {
	root := &fakeEl{tag: "body"}
	list := &fakeEl{tag: "ul", id: "menu", classes: []string{"nav", "big"}, parent: root}
	item := &fakeEl{tag: "li", classes: []string{"item"}, parent: list, state: StateHover}
	for _, c := range []struct {
		sel  string
		el   *fakeEl
		want bool
	}{
		{"li", item, true},
		{"*", item, true},
		{"LI.item", item, true},
		{"li.other", item, false},
		{"#menu", list, true},
		{"ul.nav.big#menu", list, true},
		{"body li", item, true},
		{"body > li", item, false},
		{"ul > li", item, true},
		{"body > ul > .item", item, true},
		{".nav li:hover", item, true},
		{"li:focus", item, false},
		{"body   >   ul  li", item, true},
	} {
		s, err := ParseSelector(c.sel)
		if err != nil {
			t.Errorf("%q: %v", c.sel, err)
			continue
		}
		if got := s.Matches(c.el); got != c.want {
			t.Errorf("%q matches %s: %v, want %v", c.sel, c.el.tag, got, c.want)
		}
	}
	for _, bad := range []string{"a + b", "a ~ b", "a[x]", "p::before", "a:first-child", "> a", "a >", "#", "."} {
		if _, err := ParseSelector(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestCascadeOrder(t *testing.T) {
	root := &fakeEl{tag: "body"}
	el := &fakeEl{tag: "div", id: "x", classes: []string{"c"}, parent: root, state: StateHover}
	// Specificity wins over source order (#x beats .c and div:hover);
	// div:hover (0,1,1) beats .c (0,1,0); inline comes last.
	sheet, w := ParseSheet(`
		#x { width: 1px; padding: 1px }
		div:hover { width: 2px; height: 2px; min-width: 2px }
		.c { height: 3px; min-width: 3px; max-width: 3px }
		div.c { max-width: 8px }
		.c.c { max-width: 6px }
		@media (min-width: 1px) { div { width: 9px } }
		div { margin: 4px } /* comment */
		p + q { width: 7px }
	`, 0)
	if len(w) != 2 || !strings.Contains(w[0].Error(), "@media") || !strings.Contains(w[1].Error(), "+") {
		t.Fatalf("warnings: %v", w)
	}
	s, iw := NewCascade(sheet).Compute(el, nil, "padding: 5px; float: left")
	if len(iw) != 1 {
		t.Errorf("inline warnings %v", iw)
	}
	if s.Width != Px(1) || s.Height != Px(2) || s.MinWidth != Px(2) || s.MaxWidth != Px(6) || s.Margin[0] != Px(4) || s.Padding[0] != Px(5) {
		t.Errorf("got width %v height %v min %v max %v margin %v padding %v", s.Width, s.Height, s.MinWidth, s.MaxWidth, s.Margin[0], s.Padding[0])
	}
	// Equal specificity: source order.
	sheet, _ = ParseSheet(`.c { width: 1px } div { width: 9px } .c { height: 1px } #x.c { height: 2px } div#x { height: 3px }`, 0)
	s, _ = NewCascade(sheet).Compute(el, nil, "")
	if s.Width != Px(1) || s.Height != Px(2) {
		t.Errorf("tie-break: width %v height %v", s.Width, s.Height)
	}
}

func TestSpecificity(t *testing.T) {
	for sel, want := range map[string]int{
		"*": 0, "div": 1, ".a": 1 << 8, "#a": 1 << 16, "div.a.b:hover": 3<<8 | 1,
		"ul > li#x .c:focus": 1<<16 | 2<<8 | 2, "* > *": 0,
	} {
		s, err := ParseSelector(sel)
		if err != nil || s.Specificity() != want {
			t.Errorf("%q: %x %v, want %x", sel, s.Specificity(), err, want)
		}
	}
}

func TestInheritance(t *testing.T) {
	sheet, _ := ParseSheet(`body { color: #123; font-size: 20px; text-align: center; background-color: red; text-overflow: ellipsis; opacity: .5 }`, 0)
	c := NewCascade(sheet)
	root := &fakeEl{tag: "body"}
	rs, _ := c.Compute(root, nil, "")
	ks, _ := c.Compute(&fakeEl{tag: "div", parent: root}, &rs, "")
	if ks.Color != rs.Color || ks.FontSize != geom.Px(20) || ks.TextAlign != TextAlignCenter {
		t.Errorf("inherited props not inherited: %+v", ks)
	}
	if ks.BackgroundColor != (Color{}) || ks.TextOverflow != TextOverflowClip || ks.Opacity != 1000 {
		t.Errorf("non-inherited props inherited: %+v", ks)
	}
}
