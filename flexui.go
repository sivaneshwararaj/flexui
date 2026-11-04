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
// Handlers: an element declares onclick="Name"; the Go function registered
// with Handle("Name", ...) runs when the element (or a descendant) is
// clicked. After a handler runs, the view function is called again and
// the new HTML is rendered (spec/SUBSET.md §8). See spec/STATUS.md for
// what is implemented.
package flexui

import (
	"fmt"
	"image"
	"io/fs"
	"math"
	"sync"
	"sync/atomic"

	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/render"
	"github.com/sivaneshwararaj/flexui/internal/view"
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
	// Assets resolves <img src> paths (for example an embed.FS). Without
	// it, only data: URLs load.
	Assets fs.FS
}

// Handler is a Go function bound to a handler name used in HTML, such as
// onclick="Save".
type Handler func(*Ctx)

// Ctx is passed to handlers. It describes the event and the element whose
// handler attribute is running.
type Ctx struct {
	app  *App
	ev   *view.Event
	stop bool
	// Event is the event type: "click", "keydown".
	Event string
	// ID is the id attribute of the element the handler is declared on
	// ("" if it has none).
	ID string
	// TargetID is the id of the deepest element the event is for, which
	// may be a descendant of the element with the handler.
	TargetID string
	// X, Y is the pointer position in logical pixels (pointer events).
	X, Y float64
	// Key is the key name for key events ("Enter", "a", ...).
	Key string
	// Mods holds the modifier keys of key events (platform.ModShift, ...).
	Mods uint8
}

// App returns the application the event belongs to.
func (c *Ctx) App() *App { return c.app }

// Attr returns an attribute of the element the handler is declared on,
// such as a data-id carried by a list item.
func (c *Ctx) Attr(name string) string { return c.ev.Current.Attr(name) }

// StopPropagation stops the event from reaching handlers on ancestor
// elements.
func (c *Ctx) StopPropagation() { c.stop = true }

// PreventDefault suppresses the default action of the event: for keydown,
// focus traversal (Tab) and activation (Enter, Space).
func (c *Ctx) PreventDefault() { c.ev.Prevented = true }

// App is a flexui application. It implements platform.AppHandler, so any
// backend can run it.
//
// Frames and events must come from one goroutine at a time (the platform's
// UI loop); handlers run on it. Invalidate may be called from any
// goroutine.
type App struct {
	mu       sync.Mutex
	opts     Options
	html     string
	viewFn   func() string
	css      string
	handlers map[string]Handler
	w, h     int
	view     *view.View
	dirty    atomic.Bool
	missing  map[string]bool
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
	a := &App{opts: o, handlers: map[string]Handler{}, w: o.Width, h: o.Height, view: view.New(), missing: map[string]bool{}}
	a.view.Handler = a.call
	a.view.SetAssets(o.Assets)
	a.dirty.Store(true)
	return a
}

// SetView sets a static view.
func (a *App) SetView(html, css string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.html, a.viewFn, a.css = html, nil, css
	a.dirty.Store(true)
}

// SetViewFunc sets a view that is regenerated from application state: fn
// is called again before the next frame after any handler ran (or after
// Invalidate). fn typically executes an html/template over the state. If
// it returns the same HTML as before, nothing is re-parsed.
func (a *App) SetViewFunc(fn func() string, css string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.html, a.viewFn, a.css = "", fn, css
	a.dirty.Store(true)
}

// Handle binds a handler name used in the HTML to a Go function.
// Registering the same name again replaces the previous handler.
func (a *App) Handle(name string, h Handler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handlers[name] = h
}

// Invalidate marks the view as changed, so the view function runs again
// before the next frame. Handlers do this automatically; call it after
// changing state from elsewhere (a timer, a goroutine).
func (a *App) Invalidate() { a.dirty.Store(true) }

// Config reports the surface the app wants. Backends use it in Run.
func (a *App) Config() platform.Config {
	return platform.Config{Title: a.opts.Title, Width: a.opts.Width, Height: a.opts.Height, Scale: a.opts.Scale}
}

// refresh regenerates the view source if needed. Called with mu held.
func (a *App) refresh() {
	if a.dirty.Swap(false) {
		html := a.html
		if a.viewFn != nil {
			html = a.viewFn()
		}
		a.view.SetSource(html, a.css)
	}
}

// Frame implements platform.AppHandler: it renders the current view. Only
// the stages whose inputs changed are recomputed.
func (a *App) Frame(buf *image.RGBA, w, h int, scale float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.w, a.h = w, h
	a.refresh()
	a.view.Frame(buf, w, h, scale)
}

// Warnings returns the warnings of the current view, followed by handler
// names used in the HTML that have no registered handler.
func (a *App) Warnings() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append(append([]string(nil), a.view.Warnings()...), a.warnings...)
}

// Bounds returns the border box, in logical pixels, of the first element
// with the given id in the current view (as of the last frame).
func (a *App) Bounds(id string) (x, y, w, h float64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	r, ok := a.view.Bounds(id)
	return r.X.Float(), r.Y.Float(), r.W.Float(), r.H.Float(), ok
}

func unit(f float64) geom.Unit { return geom.Unit(math.Round(f * geom.UnitsPerPx)) }

// Event implements platform.AppHandler.
func (a *App) Event(e platform.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Events act on the current view: regenerate it first if state
	// changed since the last frame.
	a.refresh()
	switch e.Kind {
	case platform.EventResize:
		a.w, a.h = e.Width, e.Height
	case platform.EventPointerMove:
		a.view.PointerMove(unit(e.X), unit(e.Y))
	case platform.EventPointerLeave:
		a.view.PointerLeave()
	case platform.EventPointerDown:
		if e.Button == 0 {
			a.view.PointerDown(unit(e.X), unit(e.Y))
		}
	case platform.EventPointerUp:
		if e.Button == 0 {
			a.view.PointerUp(unit(e.X), unit(e.Y))
		}
	case platform.EventWheel:
		a.view.Wheel(unit(e.X), unit(e.Y), unit(e.DX), unit(e.DY))
	case platform.EventKeyDown:
		a.view.KeyDown(e.Key, e.Mods, e.Mods&platform.ModShift != 0)
	case platform.EventKeyUp:
		a.view.KeyUp(e.Key)
	}
}

// Focus focuses the element with the given id ("" blurs), as if it had
// been clicked or tabbed to. It reports whether a focusable element with
// that id exists in the current view.
func (a *App) Focus(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.refresh()
	return a.view.FocusID(id)
}

// FocusedID returns the id of the focused element ("" if none).
func (a *App) FocusedID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.view.FocusedID()
}

// ScrollOffset returns how far the scroll container with the given id is
// scrolled, in logical pixels.
func (a *App) ScrollOffset(id string) (x, y float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	o := a.view.ScrollOffset(id)
	return o.X.Float(), o.Y.Float()
}

// call runs the Go handler for one element on an event's path. It is the
// view's Handler; it reports whether propagation was stopped.
func (a *App) call(ev *view.Event) bool {
	h, ok := a.handlers[ev.Name]
	if !ok {
		if !a.missing[ev.Name] {
			a.missing[ev.Name] = true
			a.warnings = append(a.warnings, fmt.Sprintf("no handler registered for %q (on%s)", ev.Name, ev.Type))
		}
		return false
	}
	c := &Ctx{app: a, ev: ev, Event: ev.Type, ID: ev.Current.ID(), TargetID: ev.Target.ID(),
		X: ev.X.Float(), Y: ev.Y.Float(), Key: ev.Key, Mods: ev.Mods}
	h(c)
	a.dirty.Store(true)
	return c.stop
}

var _ platform.AppHandler = (*App)(nil)
