package flexui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/sivaneshwararaj/flexui/platform/headless"
)

func testPNG() []byte {
	m := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			m.SetNRGBA(x, y, color.NRGBA{uint8(40 + x*8), 120, uint8(220 - y*6), 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, m)
	return b.Bytes()
}

const m5bView = `<div class="card">
  <div class="head"><img src="logo.png"><div class="title">Settings <span class="badge">beta</span></div></div>
  <p id="intro">Read the <a id="docs" onclick="Docs">documentation page</a> before changing <strong>anything</strong> here.</p>
  <div class="row">
    <input id="name" onfocus="Focus" onblur="Blur">
    <button id="save" onclick="Save" onfocus="Focus" onblur="Blur">Save</button>
    <button id="del" disabled onclick="Del">Delete</button>
  </div>
  <div id="log" class="log">{{range .}}<div class="entry">{{.}}</div>{{end}}</div>
</div>`

const m5bCSS = `
body { background-color: #f4f5f7; padding: 12px; font-size: 13px; color: #1e222a; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 8px; padding: 12px; gap: 10px; }
.head { flex-direction: row; align-items: center; gap: 8px; }
img { border-radius: 6px; }
.title { font-size: 16px; font-weight: 700; flex-direction: row; }
.badge { color: #6b7280; font-size: 11px; font-weight: 400; background-color: #eef2ff; }
a:hover { color: #0b3d91; background-color: #e8eefc; }
.row { flex-direction: row; gap: 8px; align-items: center; }
#name { width: 110px; }
#intro { width: 180px; }
.log { overflow: scroll; height: 70px; border: 1px solid #e5e7eb; border-radius: 4px; }
.entry { padding: 2px 6px; flex-shrink: 0; }
`

type m5bApp struct {
	app *App
	log []string
}

func newM5bApp(entries int) *m5bApp {
	m := &m5bApp{}
	var data []string
	for i := 1; i <= entries; i++ {
		data = append(data, fmt.Sprintf("Log entry %d", i))
	}
	m.app = New(Options{Width: 360, Height: 280, Assets: fstest.MapFS{"logo.png": {Data: testPNG()}}})
	m.app.SetViewFunc(func() string {
		var b bytes.Buffer
		templateMust(m5bView).Execute(&b, data)
		return b.String()
	}, m5bCSS)
	rec := func(name string) Handler {
		return func(c *Ctx) { m.log = append(m.log, name+":"+c.ID) }
	}
	for _, n := range []string{"Docs", "Save", "Del", "Focus", "Blur"} {
		m.app.Handle(n, rec(n))
	}
	return m
}

// TestInlineLinkAndDisabled: inline links are hit, hovered and clicked
// through their fragments; disabled buttons get no clicks or focus;
// focus and blur events; images from Options.Assets; default styles.
func TestInlineLinkAndDisabled(t *testing.T) {
	m := newM5bApp(8)
	s := headless.Start(m.app)
	if w := m.app.Warnings(); len(w) != 0 {
		t.Fatalf("warnings %q", w)
	}
	checkGolden(t, "m5b_base", s.Frame())

	// The link wraps onto two lines; Bounds is the union of its
	// fragments, and both fragments are hit.
	x, y, w, h, ok := m.app.Bounds("docs")
	if !ok || h < 30 {
		t.Errorf("link bounds %v %v %v %v: expected two lines", x, y, w, h)
	}
	s.Click(x+w-5, y+5)   // first line, right end
	s.Click(x+5, y+h-5)   // second line, left end
	s.Click(x+w-5, y+h-5) // second line, past the link's end: the paragraph
	if fmt.Sprint(m.log) != "[Docs:docs Docs:docs]" {
		t.Errorf("link clicks: %v", m.log)
	}
	s.Move(x+5, y+h-5)
	checkGolden(t, "m5b_link_hover", s.Frame())
	s.Leave()

	// Disabled button: no click, no focus, not in the Tab order.
	m.log = nil
	s.Click(center(t, m.app, "del"))
	if len(m.log) != 0 || m.app.FocusedID() != "" {
		t.Errorf("disabled button: log %v focus %q", m.log, m.app.FocusedID())
	}
	s.Key("Tab", 0) // docs link
	s.Key("Tab", 0) // input
	s.Key("Tab", 0) // save
	s.Key("Tab", 0) // wraps (delete is skipped)
	s.Click(150, 270)
	want := "[Focus:name Blur:name Focus:save Blur:save]"
	if fmt.Sprint(m.log) != want {
		t.Errorf("focus events: %v, want %v", m.log, want)
	}
	m.log = nil
	s.Click(center(t, m.app, "save"))
	checkGolden(t, "m5b_save_focused", s.Frame())
	if fmt.Sprint(m.log) != "[Focus:save Save:save]" {
		t.Errorf("save click: %v", m.log)
	}
}

// TestScrollbarDrag: the overlay scrollbar thumb can be dragged.
func TestScrollbarDrag(t *testing.T) {
	m := newM5bApp(20)
	s := headless.Start(m.app)
	x, y, w, _, _ := m.app.Bounds("log")
	// Client height 70, content 20 entries x 19px (2+2 padding + 15px
	// line at 13px... measured below) -> thumb at the right edge.
	tx, ty := x+w-1-2-3, y+1+2+5
	s.Drag(tx, ty, tx, ty+20, 4)
	_, off := m.app.ScrollOffset("log")
	if off <= 0 {
		t.Fatalf("thumb drag did not scroll: %v", off)
	}
	checkGolden(t, "m5b_scrollbar_dragged", s.Frame())
	// Dragging further than the track clamps at the end.
	s.Drag(tx, ty+20, tx, ty+400, 2)
	_, end := m.app.ScrollOffset("log")
	s.Wheel(tx-20, ty, 0, 1000)
	if _, e2 := m.app.ScrollOffset("log"); e2 != end {
		t.Errorf("thumb drag end %v, wheel end %v", end, e2)
	}
}
