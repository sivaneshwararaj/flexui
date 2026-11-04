package layout

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/geom"
	"github.com/sivaneshwararaj/flexui/style"
)

// Test trees are described with CSS declaration strings so that the same
// case can be rendered by Chromium (see scripts/crosscheck.sh).

type leafKind uint8

const (
	leafNone leafKind = iota
	leafFixed
	leafText
)

type spec struct {
	id    string
	css   string
	kids  []*spec
	leaf  leafKind
	w, h  int // fixed size, or one word's size for text leaves
	words int
}

// E is an element with children.
func E(id, css string, kids ...*spec) *spec { return &spec{id: id, css: css, kids: kids} }

// F is a leaf whose content is a fixed w x h box.
func F(id, css string, w, h int) *spec { return &spec{id: id, css: css, leaf: leafFixed, w: w, h: h} }

// T is a text-like leaf: `words` words of w x h each that wrap to the
// available width (min-content = one word, max-content = one line).
func T(id, css string, words, w, h int) *spec {
	return &spec{id: id, css: css, leaf: leafText, w: w, h: h, words: words}
}

type rect [4]float64 // x, y, w, h in px, absolute (relative to viewport)

type layoutCase struct {
	name   string
	vw, vh int
	root   *spec
	want   map[string]rect // hand-checked expectations (optional)
	// noChrome, if set, explains why Chromium's result is not used as
	// expectation (deliberate deviation); want must then be complete.
	noChrome string
}

func build(t testing.TB, s *spec) *Node {
	st := style.Initial()
	decls, w := style.ParseDeclarations(s.css)
	w = append(w, st.ApplyAll(decls, style.Context{})...)
	if len(w) != 0 {
		t.Fatalf("%s: style warnings: %v", s.id, w)
	}
	n := &Node{Style: &st}
	switch s.leaf {
	case leafFixed:
		sz := geom.Size{W: geom.Px(s.w), H: geom.Px(s.h)}
		n.Measure = func(_, _ Opt, _, _ Avail) geom.Size { return sz }
	case leafText:
		n.Measure = textMeasure(s.words, geom.Px(s.w), geom.Px(s.h))
	}
	for _, k := range s.kids {
		n.Children = append(n.Children, build(t, k))
	}
	return n
}

func textMeasure(words int, ww, wh geom.Unit) MeasureFunc {
	return func(kw, _ Opt, aw, _ Avail) geom.Size {
		per := words
		switch {
		case kw.OK:
			per = int(kw.V / ww)
		case aw.Kind == Definite:
			per = int(aw.V / ww)
		case aw.Kind == MinContent:
			per = 1
		}
		per = max(1, min(per, words))
		lines := (words + per - 1) / per
		return geom.Size{W: ww * geom.Unit(per), H: wh * geom.Unit(lines)}
	}
}

// collect returns absolute rects of all nodes with ids.
func collect(s *spec, n *Node, ox, oy geom.Unit, out map[string]rect) {
	x, y := ox+n.Box.X, oy+n.Box.Y
	if s.id != "" {
		out[s.id] = rect{x.Float(), y.Float(), n.Box.W.Float(), n.Box.H.Float()}
	}
	for i, k := range s.kids {
		collect(k, n.Children[i], x, y, out)
	}
}

func runCase(t testing.TB, c layoutCase) map[string]rect {
	root := build(t, c.root)
	Layout(root, geom.Px(c.vw), geom.Px(c.vh))
	got := map[string]rect{}
	collect(c.root, root, 0, 0, got)
	return got
}

const tolerance = 1.0 / 64 // one layout unit

func closeRect(a, b rect) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > tolerance+1e-9 {
			return false
		}
	}
	return true
}

func loadChrome(t *testing.T) map[string]map[string]rect {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "layout", "chrome.json"))
	if err != nil {
		t.Logf("no chrome.json (%v); run `make crosscheck`", err)
		return nil
	}
	var m map[string]map[string]rect
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLayoutCases(t *testing.T) {
	chrome := loadChrome(t)
	seen := map[string]bool{}
	for _, c := range cases {
		if seen[c.name] {
			t.Fatalf("duplicate case name %q", c.name)
		}
		seen[c.name] = true
		t.Run(c.name, func(t *testing.T) {
			got := runCase(t, c)
			for _, id := range sortedKeys(c.want) {
				if g, ok := got[id]; !ok || !closeRect(g, c.want[id]) {
					t.Errorf("%s: got %v, want %v (hand-written)", id, g, c.want[id])
				}
			}
			if c.noChrome != "" {
				if len(c.want) == 0 {
					t.Errorf("noChrome case needs want")
				}
				return
			}
			ref, ok := chrome[c.name]
			if !ok {
				t.Fatalf("no Chromium reference for this case; run `make crosscheck`")
			}
			for _, id := range sortedKeys(ref) {
				if g := got[id]; !closeRect(g, ref[id]) {
					t.Errorf("%s: got %v, chrome %v", id, g, ref[id])
				}
			}
		})
	}
}

func sortedKeys(m map[string]rect) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// ---- Chromium cross-check support ----

var emitHTML = flag.String("emit-html", "", "write each layout case as an HTML file into this directory")

// TestEmitHTML writes HTML files for scripts/crosscheck.sh. It does nothing
// unless -emit-html is given.
func TestEmitHTML(t *testing.T) {
	if *emitHTML == "" {
		t.Skip("use -emit-html=DIR")
	}
	if err := os.MkdirAll(*emitHTML, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if c.noChrome != "" {
			continue
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;padding:0}
#vp{position:absolute;left:0;top:0;width:%dpx;height:%dpx;display:grid;grid-template-columns:minmax(0,1fr);grid-template-rows:minmax(0,1fr)}
#vp *{display:flex;flex-direction:column;position:relative;margin:0;padding:0;border:0 solid;box-sizing:content-box;min-width:auto;min-height:auto}
</style></head><body><div id="vp">`, c.vw, c.vh)
		writeSpec(&sb, c.root)
		sb.WriteString("</div></body></html>\n")
		if err := os.WriteFile(filepath.Join(*emitHTML, c.name+".html"), []byte(sb.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeSpec(sb *strings.Builder, s *spec) {
	css := s.css
	if s.leaf == leafText {
		// Emulate wrapping words with a wrapping row of fixed boxes.
		css = "flex-direction:row;flex-wrap:wrap;align-content:flex-start;align-items:flex-start;" + css
	}
	fmt.Fprintf(sb, `<div data-id="%s" style="%s">`, html.EscapeString(s.id), html.EscapeString(css))
	switch s.leaf {
	case leafFixed:
		fmt.Fprintf(sb, `<i style="width:%dpx;height:%dpx;flex-shrink:0"></i>`, s.w, s.h)
	case leafText:
		for i := 0; i < s.words; i++ {
			fmt.Fprintf(sb, `<i style="width:%dpx;height:%dpx;flex-shrink:0"></i>`, s.w, s.h)
		}
	}
	for _, k := range s.kids {
		writeSpec(sb, k)
	}
	sb.WriteString("</div>")
}
