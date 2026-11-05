package flexui

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

var update = flag.Bool("update", false, "rewrite golden PNGs")

const counterView = `<div class="card">
  <div class="label">Count</div>
  <div id="value" class="value">{{.Count}}</div>
  <div class="row">
    <button id="dec" onclick="Dec">−</button>
    <button id="inc" onclick="Inc">+</button>
  </div>
  {{if ge .Count 3}}<div id="note" class="note">That is a lot of clicks.</div>{{end}}
</div>`

const counterCSS = `
body { background-color: #f4f5f7; color: #1e222a; font-size: 14px; align-items: center; justify-content: center; }
.card { background-color: #fff; border: 1px solid #d0d4da; border-radius: 10px; padding: 16px 24px; gap: 8px; align-items: center; min-width: 160px; }
.label { color: #6b7280; font-size: 13px; }
.value { font-size: 40px; font-weight: 700; }
.row { flex-direction: row; gap: 8px; }
button { width: 44px; height: 36px; border-radius: 8px; background-color: #3478f6; color: #fff; font-size: 20px; font-weight: 700; align-items: center; justify-content: center; }
.note { color: #b42318; font-size: 12px; }
`

type counter struct {
	Count int
	app   *App
	views int // view function calls
}

func newCounter(scale float64) *counter {
	c := &counter{}
	tmpl := template.Must(template.New("v").Parse(counterView))
	c.app = New(Options{Title: "Counter", Width: 280, Height: 220, Scale: scale})
	c.app.SetViewFunc(func() string {
		c.views++
		var b bytes.Buffer
		tmpl.Execute(&b, c)
		return b.String()
	}, counterCSS)
	c.app.Handle("Inc", func(*Ctx) { c.Count++ })
	c.app.Handle("Dec", func(*Ctx) { c.Count-- })
	return c
}

func (c *counter) html() string {
	var b bytes.Buffer
	template.Must(template.New("v").Parse(counterView)).Execute(&b, c)
	return b.String()
}

func center(t *testing.T, a *App, id string) (float64, float64) {
	t.Helper()
	x, y, w, h, ok := a.Bounds(id)
	if !ok {
		t.Fatalf("no element #%s", id)
	}
	return x + w/2, y + h/2
}

// checkGolden compares img with testdata/app/<name>.png exactly (the
// renderer is deterministic), or rewrites it with -update.
func checkGolden(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	checkGoldenAt(t, filepath.Join("testdata", "app", name+".png"), img)
}

// checkGoldenAt compares img with the PNG at path (written with -update;
// a mismatch is saved next to it in _failed/).
func checkGoldenAt(t *testing.T, path string, img *image.RGBA) {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(path), ".png")
	if *update {
		os.MkdirAll(filepath.Dir(path), 0o755)
		f, _ := os.Create(path)
		png.Encode(f, img)
		f.Close()
		return
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	defer f.Close()
	want, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if diff := pixelDiff(img, want); diff != "" {
		os.MkdirAll(filepath.Join(filepath.Dir(path), "_failed"), 0o755)
		g, _ := os.Create(filepath.Join(filepath.Dir(path), "_failed", name+".png"))
		png.Encode(g, img)
		g.Close()
		t.Errorf("%s: %s", name, diff)
	}
}

func pixelDiff(a *image.RGBA, b image.Image) string {
	if a.Bounds() != b.Bounds() {
		return fmt.Sprintf("size %v, want %v", a.Bounds(), b.Bounds())
	}
	n := 0
	for y := a.Rect.Min.Y; y < a.Rect.Max.Y; y++ {
		for x := a.Rect.Min.X; x < a.Rect.Max.X; x++ {
			r, g, bb, al := b.At(x, y).RGBA()
			p := a.RGBAAt(x, y)
			if p.R != uint8(r>>8) || p.G != uint8(g>>8) || p.B != uint8(bb>>8) || p.A != uint8(al>>8) {
				n++
			}
		}
	}
	if n > 0 {
		return fmt.Sprintf("%d pixels differ", n)
	}
	return ""
}

// TestCounterClicks: clicking buttons changes state, and the re-rendered
// frame matches the golden and a fresh render of the same state.
func TestCounterClicks(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		c := newCounter(scale)
		s := headless.Start(c.app)
		suffix := ""
		if scale == 2 {
			suffix = "@2x"
		}
		checkGolden(t, "counter_0"+suffix, s.Frame())
		for i := 0; i < 3; i++ {
			x, y := center(t, c.app, "inc")
			s.Click(x, y)
		}
		if c.Count != 3 {
			t.Fatalf("count %d after 3 clicks", c.Count)
		}
		checkGolden(t, "counter_3"+suffix, s.Frame())
		x, y := center(t, c.app, "dec")
		s.Click(x, y)
		if c.Count != 2 {
			t.Fatalf("count %d after dec", c.Count)
		}
		// The incrementally updated frame equals a fresh render (with the
		// pointer away, so no :hover, and focus cleared).
		s.Leave()
		c.app.Focus("")
		s.Send(platform.Event{Kind: platform.EventNone})
		fresh := New(Options{Width: 280, Height: 220, Scale: scale})
		fresh.SetView(c.html(), counterCSS)
		img, _ := headless.RenderImage(fresh)
		if d := pixelDiff(s.Frame(), img); d != "" {
			t.Errorf("scale %v: re-rendered frame differs from a fresh render: %s", scale, d)
		}
		if w := c.app.Warnings(); len(w) != 0 {
			t.Errorf("warnings %q", w)
		}
	}
}

// TestClickSemantics: bubbling, StopPropagation, Ctx fields, clicks that
// start and end on different elements, and missing handlers.
func TestClickSemantics(t *testing.T) {
	var log []string
	a := New(Options{Width: 200, Height: 100})
	a.SetView(`
<div id="outer" onclick="Outer" data-k="o" style="padding: 10px; flex-direction: row; gap: 10px">
  <div id="a" onclick="Inner" data-k="a" style="width: 40px; height: 40px; background-color: red"><div id="a1" style="width: 10px; height: 10px"></div></div>
  <div id="b" onclick="Stop" style="width: 40px; height: 40px; background-color: blue"></div>
  <div id="c" onclick="Nobody" style="width: 40px; height: 40px"></div>
</div>`, "")
	rec := func(name string) Handler {
		return func(c *Ctx) {
			log = append(log, fmt.Sprintf("%s:%s/%s/%s/%s", name, c.Event, c.ID, c.TargetID, c.Attr("data-k")))
		}
	}
	a.Handle("Outer", rec("Outer"))
	a.Handle("Inner", rec("Inner"))
	a.Handle("Stop", func(c *Ctx) { log = append(log, "Stop"); c.StopPropagation() })
	s := headless.Start(a)

	s.Click(center(t, a, "a1"))
	s.Click(center(t, a, "b"))
	// Press on #a, release on #b: the click goes to their common ancestor.
	ax, ay := center(t, a, "a")
	bx, by := center(t, a, "b")
	s.Press(ax, ay)
	s.Release(bx, by)
	// Outside every element with a handler (on body).
	s.Click(199, 99)
	s.Click(center(t, a, "c"))

	want := []string{
		"Inner:click/a/a1/a", "Outer:click/outer/a1/o",
		"Stop",
		"Outer:click/outer/outer/o",
		"Outer:click/outer/c/o",
	}
	if fmt.Sprint(log) != fmt.Sprint(want) {
		t.Errorf("handlers ran:\n got %q\nwant %q", log, want)
	}
	if w := a.Warnings(); len(w) != 1 || w[0] != `no handler registered for "Nobody" (onclick)` {
		t.Errorf("warnings %q", w)
	}
	if w := CheckHTML(`<div onmouseover="X"></div>`, ""); len(w) != 1 {
		t.Errorf("unsupported handler attribute: %q", w)
	}
}

// TestFrameCaching: frames only redo the stages whose inputs changed.
func TestFrameCaching(t *testing.T) {
	c := newCounter(1)
	s := headless.Start(c.app)
	st := c.app.view.Stats()
	if st.Parses != 1 || st.Render.Styles != 1 || st.Render.Layouts != 1 || st.Render.Paints != 1 {
		t.Fatalf("first frame: %+v", st)
	}
	shapes := st.Render.TextShapes

	// Pointer movement over elements no :hover rule can match: no view
	// call, no parse, no paint; frames are copied.
	views := c.views
	s.Move(1, 1)
	s.Move(2, 2)
	st = c.app.view.Stats()
	if c.views != views || st.Parses != 1 || st.Render.Paints != 1 || st.Copies < 2 {
		t.Errorf("idle frames did work: views %d->%d %+v", views, c.views, st)
	}

	// A click that changes the count: the view function runs once more,
	// the document is reparsed once, and only the changed text is shaped.
	// (Hover, press and focus of the button restyle and repaint: the
	// default style sheet styles button:hover/:active/:focus.)
	s.Click(center(t, c.app, "inc"))
	st = c.app.view.Stats()
	if st.Parses != 2 {
		t.Errorf("after click: %+v", st)
	}
	if n := st.Render.TextShapes - shapes; n != 1 { // only the "1"
		t.Errorf("click reshaped %d text runs, want 1", n)
	}
	if st.Render.SheetParses != 1 {
		t.Errorf("style sheet parsed %d times", st.Render.SheetParses)
	}

	// A handler that does not change the HTML: the view function runs,
	// but nothing is reparsed and no text is reshaped. Pressing an already
	// hovered and focused button repaints only for :active (on and off).
	c.app.Handle("Inc", func(*Ctx) {})
	s.Click(center(t, c.app, "inc"))
	st2 := c.app.view.Stats()
	if st2.Parses != st.Parses || st2.Render.TextShapes != st.Render.TextShapes || st2.Render.Paints != st.Render.Paints+2 {
		t.Errorf("unchanged view did work: %+v (before %+v)", st2, st)
	}

	// Resize: layout and paint again, no restyle.
	s.Resize(300, 240)
	st3 := c.app.view.Stats()
	if st3.Render.Styles != st2.Render.Styles || st3.Render.Layouts != st2.Render.Layouts+1 || st3.Render.Paints != st2.Render.Paints+1 {
		t.Errorf("resize: %+v (before %+v)", st3, st2)
	}
}

// TestInvalidate: state changed outside a handler shows after Invalidate.
func TestInvalidate(t *testing.T) {
	c := newCounter(1)
	s := headless.Start(c.app)
	c.Count = 7
	s.Send(platform.Event{Kind: platform.EventNone})
	if c.views != 1 {
		t.Fatalf("view re-run without Invalidate")
	}
	c.app.Invalidate()
	s.Send(platform.Event{Kind: platform.EventNone})
	if c.views != 2 {
		t.Fatalf("view not re-run after Invalidate")
	}
}

func BenchmarkCounterClick(b *testing.B) {
	c := newCounter(1)
	s := headless.Start(c.app)
	x, y, w, h, _ := c.app.Bounds("inc")
	for i := 0; i < b.N; i++ {
		s.Click(x+w/2, y+h/2)
	}
}

func BenchmarkIdleFrame(b *testing.B) {
	c := newCounter(1)
	s := headless.Start(c.app)
	for i := 0; i < b.N; i++ {
		s.Move(1, 1)
	}
}
