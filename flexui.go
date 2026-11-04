// Package flexui is a portable desktop UI library: views are written in
// HTML plus a strict, flexbox-only CSS subset (see spec/SUBSET.md), and
// interaction logic is written in Go. There is no JavaScript and no
// browser engine; rendering is done by a pure-Go software renderer, so the
// same input produces the same pixels on every platform.
//
// A typical program:
//
//	app := flexui.New(flexui.Options{Title: "Hello", Width: 800, Height: 600})
//	app.SetView(viewHTML, styleCSS)
//	app.Handle("Save", func(c *flexui.Ctx) { /* update state */ })
//	headless.Render(app, "out.png") // or darwin.Run(app)
//
// Backends live in separate packages under platform/ so that importing
// flexui never compiles code for other operating systems.
//
// Status: views render (M3) except text (M4); handlers run from M5. See
// spec/STATUS.md.
package flexui

import (
	"image"

	"github.com/sivaneshwararaj/flexui/internal/render"
	"github.com/sivaneshwararaj/flexui/platform"
)

// RenderHTML renders an HTML document styled with css (plus any <style>
// elements in it) at w x h logical pixels, scale 1, on a white canvas.
// Unsupported HTML or CSS is ignored; use CheckHTML to see warnings.
func RenderHTML(html, css string, w, h int) *image.RGBA {
	img, _ := render.Render(html, css, w, h, 1)
	return img
}

// CheckHTML returns the warnings produced while rendering html and css:
// unsupported elements, properties, values and selectors.
func CheckHTML(html, css string) []string {
	d := render.Build(html, css)
	return d.Warnings
}

// Options configures an App.
type Options struct {
	Title  string
	Width  int // initial window width in logical pixels
	Height int // initial window height in logical pixels
	// Scale is device pixels per logical pixel. 0 lets the platform decide.
	Scale float64
}

// Handler is a Go function bound to a handler name used in HTML, such as
// onclick="Save".
type Handler func(*Ctx)

// Ctx is passed to handlers. It describes the event and the element that
// triggered it.
type Ctx struct {
	app *App
	// ID is the id attribute of the element the handler is declared on
	// ("" if it has none).
	ID string
}

// App returns the application the event belongs to.
func (c *Ctx) App() *App { return c.app }

// App is a flexui application. It implements platform.AppHandler, so any
// backend can run it.
type App struct {
	opts     Options
	html     string
	viewFn   func() string
	css      string
	handlers map[string]Handler
	w, h     int
	warnings []string
}

// New creates an application.
func New(o Options) *App {
	if o.Width <= 0 {
		o.Width = 800
	}
	if o.Height <= 0 {
		o.Height = 600
	}
	return &App{opts: o, handlers: map[string]Handler{}, w: o.Width, h: o.Height}
}

// SetView sets a static view.
func (a *App) SetView(html, css string) {
	a.html, a.viewFn, a.css = html, nil, css
}

// SetViewFunc sets a view that is regenerated from application state after
// every handler call. fn typically closes over the state and executes an
// html/template.
func (a *App) SetViewFunc(fn func() string, css string) {
	a.html, a.viewFn, a.css = "", fn, css
}

// Handle binds a handler name used in the HTML to a Go function.
// Registering the same name again replaces the previous handler.
func (a *App) Handle(name string, h Handler) { a.handlers[name] = h }

// Config reports the surface the app wants. Backends use it in Run.
func (a *App) Config() platform.Config {
	return platform.Config{Title: a.opts.Title, Width: a.opts.Width, Height: a.opts.Height, Scale: a.opts.Scale}
}

// Frame implements platform.AppHandler: it renders the current view.
func (a *App) Frame(buf *image.RGBA, w, h int, scale float64) {
	a.w, a.h = w, h
	html := a.html
	if a.viewFn != nil {
		html = a.viewFn()
	}
	a.warnings = render.RenderInto(buf, html, a.css, w, h, scale)
}

// Warnings returns the warnings from the most recent frame.
func (a *App) Warnings() []string { return a.warnings }

// Event implements platform.AppHandler.
func (a *App) Event(e platform.Event) {
	if e.Kind == platform.EventResize {
		a.w, a.h = e.Width, e.Height
	}
}

var _ platform.AppHandler = (*App)(nil)
