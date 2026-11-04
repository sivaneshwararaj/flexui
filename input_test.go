package flexui

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/platform"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

const formView = `<div class="form">
  <div class="field"><div class="label">Name</div>
    <input id="name" value="{{.Name}}" placeholder="Your name" oninput="NameInput" onchange="Changed"></div>
  <div class="field"><div class="label">Email</div>
    <input id="email" placeholder="you@example.com" onchange="Changed"></div>
  <div class="field"><div class="label">Locked</div><input id="locked" value="read only" disabled></div>
  <div class="row"><div id="echo">Hello, {{if .Name}}{{.Name}}{{else}}stranger{{end}}</div><button id="reset" onclick="Reset">Reset</button></div>
</div>`

const formCSS = `
body { background-color: #f4f5f7; padding: 12px; font-size: 13px; color: #1e222a; }
.form { background-color: #fff; border: 1px solid #d0d4da; border-radius: 8px; padding: 12px; gap: 8px; }
.field { flex-direction: row; align-items: center; gap: 8px; }
.label { width: 50px; color: #6b7280; }
.row { flex-direction: row; align-items: center; justify-content: space-between; }
`

type formApp struct {
	Name string
	app  *App
	log  []string
}

func newFormApp() *formApp {
	f := &formApp{}
	tmpl := templateMust(formView)
	f.app = New(Options{Width: 300, Height: 190})
	f.app.SetViewFunc(func() string {
		var b bytes.Buffer
		tmpl.Execute(&b, f)
		return b.String()
	}, formCSS)
	f.app.Handle("NameInput", func(c *Ctx) { f.Name = c.Value; f.log = append(f.log, "input:"+c.Value) })
	f.app.Handle("Changed", func(c *Ctx) { f.log = append(f.log, "change:"+c.ID+"="+c.Value) })
	f.app.Handle("Reset", func(c *Ctx) { f.Name = "Reset"; f.log = append(f.log, "reset") })
	return f
}

func sel(a *App, id string) string {
	c, an := a.view.InputSelection(id)
	return fmt.Sprintf("%d/%d", c, an)
}

// TestInputEditing: typing, caret and word movement, selection, deletion,
// clipboard, input/change events, controlled re-rendering.
func TestInputEditing(t *testing.T) {
	f := newFormApp()
	s := headless.Start(f.app)
	if w := f.app.Warnings(); len(w) != 0 {
		t.Fatalf("warnings %q", w)
	}
	checkGolden(t, "input_base", s.Frame())

	// Click into the empty name field and type.
	s.Click(center(t, f.app, "name"))
	s.Type("Ada")
	s.Type(" Lovelace")
	if f.Name != "Ada Lovelace" || f.app.Value("name") != "Ada Lovelace" || sel(f.app, "name") != "12/12" {
		t.Fatalf("after typing: state %q value %q sel %s", f.Name, f.app.Value("name"), sel(f.app, "name"))
	}
	s.Leave()
	checkGolden(t, "input_typed", s.Frame())

	// Word movement and selection: Ctrl+Left to "Lovelace", Shift+End.
	s.Key("ArrowLeft", platform.ModCtrl)
	if sel(f.app, "name") != "4/4" {
		t.Errorf("word left: %s", sel(f.app, "name"))
	}
	s.Key("End", platform.ModShift)
	checkGolden(t, "input_selected", s.Frame())
	s.Key("c", platform.ModMeta) // copy "Lovelace"
	if c, _ := s.Clipboard.ReadText(); c != "Lovelace" {
		t.Errorf("clipboard %q", c)
	}
	s.Key("Backspace", 0) // delete the selection
	s.Key("Backspace", platform.ModAlt)
	s.Type("Grace ")
	s.Key("v", platform.ModCtrl)
	if f.Name != "Grace Lovelace" {
		t.Errorf("after edits: %q", f.Name)
	}
	s.Key("Home", 0)
	s.Key("Delete", 0)
	s.Key("ArrowRight", platform.ModShift)
	s.Type("G") // replaces the selected "r"
	if f.Name != "Gace Lovelace" || sel(f.app, "name") != "1/1" {
		t.Errorf("home/delete/replace: %q %s", f.Name, sel(f.app, "name"))
	}

	// Tab to email: change fires for name (before blur); paste there.
	f.log = nil
	s.Key("Tab", 0)
	s.Key("v", platform.ModCtrl)
	s.Key("Enter", 0) // change on Enter
	s.Key("Enter", 0) // unchanged: no second change
	want := "[change:name=Gace Lovelace change:email=Lovelace]"
	if fmt.Sprint(f.log) != want {
		t.Errorf("events %v, want %v", f.log, want)
	}

	// The disabled input cannot be focused or edited.
	s.Click(center(t, f.app, "locked"))
	s.Type("x")
	if f.app.Value("locked") != "read only" || f.app.FocusedID() == "locked" {
		t.Errorf("disabled input edited: %q focus %q", f.app.Value("locked"), f.app.FocusedID())
	}

	// A programmatic change of the value replaces the text.
	s.Click(center(t, f.app, "reset"))
	if f.app.Value("name") != "Reset" {
		t.Errorf("value after reset: %q", f.app.Value("name"))
	}
}

// TestInputScrollAndPointer: long text scrolls to keep the caret visible;
// clicks place the caret; dragging selects.
func TestInputScrollAndPointer(t *testing.T) {
	f := newFormApp()
	s := headless.Start(f.app)
	x, y, w, h, _ := f.app.Bounds("email")
	s.Click(x+w/2, y+h/2)
	s.Type("a.very.long.address@subdomain.example.com")
	s.Leave()
	checkGolden(t, "input_scrolled_end", s.Frame())
	s.Key("Home", 0)
	checkGolden(t, "input_scrolled_home", s.Frame())
	// Click near the left edge of the text: caret near the start; drag to
	// select the first word ("a.very" ends at index 6).
	cx := x + 1 + 6 // border + padding
	s.Click(cx+1, y+h/2)
	if c, _ := f.app.view.InputSelection("email"); c != 0 {
		t.Errorf("click at the left: caret %d", c)
	}
	end, _ := f.app.view.Document().InputCaretX(nodeByID(f.app, "email"), f.app.Value("email"), 6)
	s.Drag(cx+1, y+h/2, cx+end.Float()+0.4, y+h/2, 3)
	if sel(f.app, "email") != "6/0" {
		t.Errorf("drag selection %s", sel(f.app, "email"))
	}
	s.Leave()
	checkGolden(t, "input_drag_selected", s.Frame())
}

func nodeByID(a *App, id string) *dom.Node {
	return a.view.Document().Dom.Root.FindID(id)
}
