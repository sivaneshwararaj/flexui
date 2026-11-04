package flexui

import (
	"bytes"
	"fmt"
	"html/template"
	"image"
	"testing"

	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

const listView = `<div class="toolbar">
  <button id="add" onclick="Add">Add</button>
  <button id="clear" onclick="Clear">Clear</button>
  <a id="help" onclick="Help">Help</a>
  <div class="spacer"></div>
  <div id="status">{{len .Items}} items{{if .Sel}}, selected {{.Sel}}{{end}}</div>
</div>
<div id="list" class="list">
{{range .Items}}<div class="row{{if eq . $.Sel}} sel{{end}}" onclick="Select" data-name="{{.}}">{{.}}</div>
{{end}}</div>`

const listCSS = `
body { background-color: #f4f5f7; color: #1e222a; font-size: 13px; padding: 10px; gap: 8px; }
.toolbar { flex-direction: row; gap: 6px; align-items: center; }
.spacer { flex-grow: 1; }
button, a { padding: 4px 10px; border: 2px solid #c9ced6; border-radius: 6px; background-color: #fff; }
button:hover, a:hover { background-color: #e8eefc; }
button:active { background-color: #3478f6; color: #fff; }
button:focus, a:focus { border-color: #3478f6; }
.list { overflow: scroll; height: 150px; background-color: #fff; border: 1px solid #d0d4da; border-radius: 6px; }
.row { padding: 6px 10px; border-width: 0 0 1px 0; border-color: #eef0f3; flex-shrink: 0; }
.row:hover { background-color: #f1f5ff; }
.sel { background-color: #dbe6ff; font-weight: 700; }
`

type listApp struct {
	Items []string
	Sel   string
	app   *App
	log   []string
}

func newListApp(n int) *listApp {
	l := &listApp{}
	for i := 1; i <= n; i++ {
		l.Items = append(l.Items, fmt.Sprintf("Item %02d", i))
	}
	tmpl := template.Must(template.New("v").Parse(listView))
	l.app = New(Options{Width: 300, Height: 220})
	l.app.SetViewFunc(func() string {
		var b bytes.Buffer
		tmpl.Execute(&b, l)
		return b.String()
	}, listCSS)
	l.app.Handle("Add", func(c *Ctx) {
		l.Items = append(l.Items, fmt.Sprintf("Item %02d", len(l.Items)+1))
		l.log = append(l.log, "Add")
	})
	l.app.Handle("Clear", func(c *Ctx) { l.Sel = ""; l.log = append(l.log, "Clear") })
	l.app.Handle("Help", func(c *Ctx) { l.log = append(l.log, "Help") })
	l.app.Handle("Select", func(c *Ctx) { l.Sel = c.Attr("data-name"); l.log = append(l.log, "Select "+l.Sel) })
	return l
}

// TestHoverActiveFocusStyles: :hover, :active and :focus styles follow
// synthetic pointer input, and leaving restores the original frame.
func TestHoverActiveFocusStyles(t *testing.T) {
	l := newListApp(12)
	s := headless.Start(l.app)
	base := clone(s.Frame())
	checkGolden(t, "list_base", s.Frame())

	// :hover on the Add button and on a row. The hover change repaints
	// only damaged rectangles (and, with the damage check on in these
	// tests, is verified against a full repaint).
	before := l.app.view.Stats()
	s.Move(center(t, l.app, "add"))
	checkGolden(t, "list_hover_add", s.Frame())
	if st := l.app.view.Stats(); st.Damaged != before.Damaged+1 || st.DamagePixel-before.DamagePixel > 300*220/4 {
		t.Errorf("hover repaint: %d damaged frames (+%d px)", st.Damaged-before.Damaged, st.DamagePixel-before.DamagePixel)
	}
	s.Leave()
	if d := pixelDiff(s.Frame(), base); d != "" {
		t.Errorf("after leaving, frame differs from the base: %s", d)
	}

	// :active while pressed (and :focus, since buttons focus on press).
	x, y := center(t, l.app, "clear")
	s.Move(x, y)
	s.Press(x, y)
	checkGolden(t, "list_active_clear", s.Frame())
	s.Release(x, y)
	if l.app.FocusedID() != "clear" {
		t.Errorf("focus after click: %q", l.app.FocusedID())
	}
	// Clicking a non-focusable area blurs.
	s.Click(150, 215)
	if l.app.FocusedID() != "" {
		t.Errorf("focus after clicking body: %q", l.app.FocusedID())
	}
	s.Leave()
	if d := pixelDiff(s.Frame(), base); d != "" {
		t.Errorf("after blur and leave, frame differs from the base: %s", d)
	}
}

func clone(img *image.RGBA) *image.RGBA {
	c := *img
	c.Pix = append([]uint8(nil), img.Pix...)
	return &c
}

// TestKeyboardFocus: Tab order, wrap-around, Shift+Tab, Enter and Space
// activation, keydown handlers and PreventDefault, focus kept across
// re-render.
func TestKeyboardFocus(t *testing.T) {
	l := newListApp(3)
	s := headless.Start(l.app)
	var order []string
	for i := 0; i < 4; i++ {
		s.Key("Tab", 0)
		order = append(order, l.app.FocusedID())
	}
	if fmt.Sprint(order) != "[add clear help add]" {
		t.Errorf("tab order %v", order)
	}
	checkGolden(t, "list_focus_add", s.Frame())
	s.Key("Tab", platform.ModShift)
	if f := l.app.FocusedID(); f != "help" {
		t.Errorf("shift+tab from add: %q", f)
	}
	s.Key("Enter", 0) // activates the link
	s.Key("Tab", platform.ModShift)
	s.Key(" ", 0) // Space on "clear"
	s.Key("Tab", platform.ModShift)
	s.Key("Enter", 0) // Enter on "add": adds an item and re-renders
	if fmt.Sprint(l.log) != "[Help Clear Add]" || len(l.Items) != 4 {
		t.Errorf("activation log %v, %d items", l.log, len(l.Items))
	}
	if f := l.app.FocusedID(); f != "add" {
		t.Errorf("focus not kept across re-render: %q", f)
	}
	checkGolden(t, "list_focus_after_add", s.Frame())

	// A keydown handler can prevent Tab navigation.
	a := New(Options{Width: 100, Height: 50})
	a.SetView(`<div onkeydown="K"><button id="x">x</button><button id="y">y</button></div>`, "")
	var keys []string
	a.Handle("K", func(c *Ctx) {
		keys = append(keys, c.Key)
		if c.Key == "Tab" {
			c.PreventDefault()
		}
	})
	s2 := headless.Start(a)
	if !a.Focus("x") {
		t.Fatal("Focus(x) failed")
	}
	s2.Key("Tab", 0)
	s2.Key("q", 0)
	if a.FocusedID() != "x" || fmt.Sprint(keys) != "[Tab q]" {
		t.Errorf("prevented tab: focus %q keys %v", a.FocusedID(), keys)
	}
	// tabindex: positive first, -1 skipped, display:none skipped.
	b := New(Options{Width: 100, Height: 50})
	b.SetView(`<div id="p" tabindex="0">p</div><button id="q" tabindex="-1">q</button>
<button id="r">r</button><div id="s" tabindex="2">s</div><div id="t" tabindex="1">t</div>
<button id="u" style="display: none">u</button>`, "")
	s3 := headless.Start(b)
	order = nil
	for i := 0; i < 5; i++ {
		s3.Key("Tab", 0)
		order = append(order, b.FocusedID())
	}
	if fmt.Sprint(order) != "[t s p r t]" {
		t.Errorf("tabindex order %v", order)
	}
}

// TestScrolling: wheel and drag scrolling with clamping and clipping,
// clicks on scrolled content, drags that do not click, and the scroll
// position kept across re-renders.
func TestScrolling(t *testing.T) {
	l := newListApp(30)
	s := headless.Start(l.app)
	lx, ly := center(t, l.app, "list")
	s.Wheel(lx, ly, 0, 100)
	if _, y := l.app.ScrollOffset("list"); y != 100 {
		t.Fatalf("scroll after wheel: %v", y)
	}
	s.Leave()
	checkGolden(t, "list_scroll_100", s.Frame())

	// Clamped at the end: 30 rows of 29px (6+6 padding, 16px line, 1px
	// border) = 870px of content in a 150px client area (height: 150px is
	// the content box).
	s.Wheel(lx, ly, 0, 5000)
	_, max := l.app.ScrollOffset("list")
	if max != 870-150 {
		t.Errorf("max scroll %v, want %v", max, 870-150)
	}
	s.Wheel(lx, ly, 0, -5000)
	if _, y := l.app.ScrollOffset("list"); y != 0 {
		t.Errorf("scroll after wheel up: %v", y)
	}

	// Drag scrolling: pull content up by 58px (two rows); no click.
	s.Drag(lx, ly+40, lx, ly-18, 4)
	if _, y := l.app.ScrollOffset("list"); y != 58 {
		t.Errorf("scroll after drag: %v", y)
	}
	if len(l.log) != 0 {
		t.Errorf("drag clicked: %v", l.log)
	}

	// A click on the scrolled list hits the row now under the pointer:
	// the list top is at 10 + toolbar; the first visible row is Item 03.
	_, top, _, _, _ := l.app.Bounds("list")
	s.Click(lx, top+1+29/2)
	if l.Sel != "Item 03" {
		t.Errorf("clicked %q, want Item 03", l.Sel)
	}
	// The click re-rendered the view; the scroll position is kept.
	if _, y := l.app.ScrollOffset("list"); y != 58 {
		t.Errorf("scroll after re-render: %v", y)
	}
	s.Leave()
	checkGolden(t, "list_scrolled_selected", s.Frame())

	// A wheel over a non-scrolling element does nothing.
	ax, ay := center(t, l.app, "add")
	s.Wheel(ax, ay, 0, 50)
	if _, y := l.app.ScrollOffset("list"); y != 58 {
		t.Errorf("wheel outside the list scrolled it: %v", y)
	}
}

// TestScrollChaining: a nested scroll container passes on what it cannot
// absorb.
func TestScrollChaining(t *testing.T) {
	a := New(Options{Width: 200, Height: 200})
	a.SetView(`<div id="outer" style="overflow: scroll; height: 100px">
  <div style="height: 50px; flex-shrink: 0"></div>
  <div id="inner" style="overflow: scroll; height: 60px; flex-shrink: 0"><div style="height: 100px; flex-shrink: 0"></div></div>
  <div style="height: 100px; flex-shrink: 0"></div>
</div>`, "")
	s := headless.Start(a)
	x, y := center(t, a, "inner")
	s.Wheel(x, y, 0, 70) // inner can take 40, outer the remaining 30
	_, iy := a.ScrollOffset("inner")
	_, oy := a.ScrollOffset("outer")
	if iy != 40 || oy != 30 {
		t.Errorf("chained scroll: inner %v outer %v", iy, oy)
	}
}

func templateMust(src string) *template.Template { return template.Must(template.New("v").Parse(src)) }
