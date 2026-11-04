package style

import (
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/geom"
)

func parse(t *testing.T, css string) (Style, []Warning) {
	t.Helper()
	s := Initial()
	decls, w1 := ParseDeclarations(css)
	w2 := s.ApplyAll(decls, Context{})
	return s, append(w1, w2...)
}

func TestParseLengths(t *testing.T) {
	cases := []struct {
		css  string
		get  func(Style) Length
		want Length
	}{
		{"width: 10px", func(s Style) Length { return s.Width }, Px(10)},
		{"width: 10.5px", func(s Style) Length { return s.Width }, Units(geom.Px(10) + 32)},
		{"width: 0", func(s Style) Length { return s.Width }, Px(0)},
		{"width: 50%", func(s Style) Length { return s.Width }, Percent(50)},
		{"width: 33.333%", func(s Style) Length { return s.Width }, Length{LengthPercent, 33333}},
		{"width: 2rem", func(s Style) Length { return s.Width }, Px(32)},
		{"width: AUTO", func(s Style) Length { return s.Width }, Auto},
		{"max-width: none", func(s Style) Length { return s.MaxWidth }, None},
		{"margin-left: -4px", func(s Style) Length { return s.Margin[Left] }, Px(-4)},
		{"flex-basis: 25%", func(s Style) Length { return s.FlexBasis }, Percent(25)},
	}
	for _, c := range cases {
		s, w := parse(t, c.css)
		if len(w) != 0 {
			t.Errorf("%q: unexpected warnings %v", c.css, w)
		}
		if got := c.get(s); got != c.want {
			t.Errorf("%q: got %+v want %+v", c.css, got, c.want)
		}
	}
}

func TestParseShorthands(t *testing.T) {
	s, w := parse(t, "margin: 1px 2px 3px; padding: 4px 5px; border-width: 1px 2px 3px 4px; gap: 6px 7px")
	if len(w) != 0 {
		t.Fatal(w)
	}
	if s.Margin != [4]Length{Px(1), Px(2), Px(3), Px(2)} {
		t.Errorf("margin %+v", s.Margin)
	}
	if s.Padding != [4]Length{Px(4), Px(5), Px(4), Px(5)} {
		t.Errorf("padding %+v", s.Padding)
	}
	if s.Border != [4]geom.Unit{geom.Px(1), geom.Px(2), geom.Px(3), geom.Px(4)} {
		t.Errorf("border %+v", s.Border)
	}
	if s.RowGap != Px(6) || s.ColumnGap != Px(7) {
		t.Errorf("gap %+v %+v", s.RowGap, s.ColumnGap)
	}
}

func TestParseFlexShorthand(t *testing.T) {
	cases := []struct {
		v      string
		g, s   Factor
		basis  Length
		hasErr bool
	}{
		{"1", 1000, 1000, Px(0), false},
		{"2 3", 2000, 3000, Px(0), false},
		{"1 100px", 1000, 1000, Px(100), false},
		{"0 0 auto", 0, 0, Auto, false},
		{"none", 0, 0, Auto, false},
		{"auto", 1000, 1000, Auto, false},
		{"50%", 1000, 1000, Percent(50), false},
		{"0.5", 500, 1000, Px(0), false},
		{"1 2 3 4", 0, 0, Length{}, true},
		{"banana", 0, 0, Length{}, true},
	}
	for _, c := range cases {
		s, w := parse(t, "flex: "+c.v)
		if c.hasErr {
			if len(w) == 0 {
				t.Errorf("flex: %s: expected warning", c.v)
			}
			continue
		}
		if len(w) != 0 || s.FlexGrow != c.g || s.FlexShrink != c.s || s.FlexBasis != c.basis {
			t.Errorf("flex: %s: got %d %d %+v (%v)", c.v, s.FlexGrow, s.FlexShrink, s.FlexBasis, w)
		}
	}
}

func TestUnsupportedWarns(t *testing.T) {
	for _, css := range []string{
		"float: left",
		"display: block",
		"width: calc(1px + 2px)",
		"width: 10em",
		"justify-content: baseline",
		"align-items: baseline",
		"padding: -1px",
		"width: inherit",
		"color red",
	} {
		_, w := parse(t, css)
		if len(w) == 0 {
			t.Errorf("%q: expected a warning", css)
		}
	}
	s, w := parse(t, "width: 10px !important; /* c */ height: 5px")
	if len(w) != 1 || !strings.Contains(w[0].Error(), "important") || s.Width != Px(10) || s.Height != Px(5) {
		t.Errorf("important handling: %+v %v", s, w)
	}
}

func TestInvalidLeavesValue(t *testing.T) {
	s, _ := parse(t, "width: 10px; width: bogus")
	if s.Width != Px(10) {
		t.Errorf("invalid declaration overwrote value: %+v", s.Width)
	}
}
