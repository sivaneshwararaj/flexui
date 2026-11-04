package flexui_test

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/sivaneshwararaj/flexui"
	"github.com/sivaneshwararaj/flexui/platform/headless"
)

// The examples below are the snippets of docs/GUIDE.md; `go test` checks
// that they compile and produce the shown output.

// RenderHTML renders one static page; CheckHTML lists what is unsupported.
func ExampleRenderHTML() {
	img := flexui.RenderHTML(`<div style="width: 50px; height: 20px; background-color: #3478f6"></div>`, "", 100, 40)
	fmt.Println(img.Bounds().Dx(), img.Bounds().Dy(), img.RGBAAt(10, 10))
	fmt.Println(flexui.CheckHTML(`<table></table>`, `div { float: left }`))
	// Output:
	// 100 40 {52 120 246 255}
	// [<table> is not supported; it behaves as <div> css: float: left: unsupported property]
}

// A stateful app: the view is a function of the state, handlers change
// the state, and the view is rendered again. headless.Session drives it
// with synthetic input, as a test would.
func ExampleApp() {
	type state struct{ Count int }
	st := &state{}
	tmpl := template.Must(template.New("view").Parse(
		`<div class="row"><button id="inc" onclick="Inc">+</button><div id="n">{{.Count}}</div></div>`))

	app := flexui.New(flexui.Options{Title: "Counter", Width: 200, Height: 80})
	app.SetViewFunc(func() string {
		var b bytes.Buffer
		tmpl.Execute(&b, st)
		return b.String()
	}, `.row { flex-direction: row; gap: 8px; padding: 8px; align-items: center; }`)
	app.Handle("Inc", func(c *flexui.Ctx) { st.Count++ })

	s := headless.Start(app)
	x, y, w, h, _ := app.Bounds("inc")
	s.Click(x+w/2, y+h/2)
	s.Click(x+w/2, y+h/2)
	fmt.Println(st.Count, app.Warnings())
	// Output: 2 []
}

// Inputs: oninput gives the text in Ctx.Value; keep it in the state.
func ExampleCtx_value() {
	name := ""
	app := flexui.New(flexui.Options{Width: 240, Height: 60})
	app.SetViewFunc(func() string {
		return `<input id="name" value="` + template.HTMLEscapeString(name) + `" oninput="Name">`
	}, "")
	app.Handle("Name", func(c *flexui.Ctx) { name = c.Value })

	s := headless.Start(app)
	x, y, w, h, _ := app.Bounds("name")
	s.Click(x+w/2, y+h/2)
	s.Type("Ada")
	fmt.Println(name, app.Value("name"))
	// Output: Ada Ada
}

// Uncontrolled inputs: no oninput handler; a handler reads the text with
// Ctx.InputValue (App methods must not be called inside handlers).
func ExampleCtx_InputValue() {
	var saved string
	app := flexui.New(flexui.Options{Width: 300, Height: 60})
	app.SetView(`<div style="flex-direction: row; gap: 8px"><input id="title"><button id="save" onclick="Save">Save</button></div>`, "")
	app.Handle("Save", func(c *flexui.Ctx) { saved = c.InputValue("title") })

	s := headless.Start(app)
	x, y, w, h, _ := app.Bounds("title")
	s.Click(x+w/2, y+h/2)
	s.Type("Draft 1")
	x, y, w, h, _ = app.Bounds("save")
	s.Click(x+w/2, y+h/2)
	fmt.Println(saved)
	// Output: Draft 1
}
