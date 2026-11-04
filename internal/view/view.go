// Package view keeps a rendered document alive between frames: it
// re-renders only the stages that changed (parse, style, layout, place,
// paint), does hit testing, and turns pointer and key input into DOM-like
// events (click, ...) dispatched to named handlers (spec/SUBSET.md §8).
//
// A View is not safe for concurrent use; the app runtime serializes
// frames and events.
package view

import (
	"image"
	"io/fs"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/render"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Stats counts the work done, for tests and tuning.
type Stats struct {
	Frames int // Frame calls
	Parses int // HTML parses
	Copies int // frames served from the retained image without painting
	Render render.Stats
}

// View is a live document.
type View struct {
	// Handler is called for each element on an event's propagation path
	// that declares a handler for it (e.g. onclick="Save"). It returns
	// true to stop propagation.
	Handler func(ev *Event) bool

	html, css string
	cache     *render.Cache
	parsed    *render.Parsed
	doc       *render.Document

	w, h  int
	scale float64

	needStyle, needLayout, needPlace, needPaint bool

	frame  *image.RGBA // retained output of the last paint
	list   paint.List
	raster paint.Raster

	// Pointer state.
	down       *dom.Node // element under the pointer at pointer down
	downAt     geom.Point
	pressed    bool
	pointer    geom.Point // last known pointer position
	hasPointer bool
	dragging   *dom.Node // scroll container being dragged, if any
	dragLast   geom.Point
	thumb      *thumbDrag // scrollbar thumb being dragged, if any

	// Interaction state (spec/SUBSET.md §8.3-8.5).
	hover, active, focus *dom.Node
	focusKey             string // key of the focused element, kept across rebuilds
	hoverStale           bool   // hover must be recomputed after the next placement
	space                *dom.Node
	scroll               map[string]geom.Point // scroll offsets by element key
	inputs               map[string]*inputState
	selecting            *dom.Node // input whose text is being selected by dragging
	clip                 Clipboard

	stats Stats
}

// New returns an empty view.
func New() *View {
	return &View{cache: render.NewCache(), needStyle: true, scroll: map[string]geom.Point{},
		inputs: map[string]*inputState{}, clip: &memClipboard{}}
}

// SetAssets sets the file system <img src> paths are read from.
func (v *View) SetAssets(fsys fs.FS) {
	v.cache.Assets = fsys
	v.needStyle = true
}

// SetSource sets the HTML and CSS of the view. Unchanged strings cost
// nothing; a changed CSS restyles, a changed HTML reparses.
func (v *View) SetSource(html, css string) {
	if v.parsed == nil || html != v.html {
		v.html, v.parsed = html, nil
		v.needStyle = true
	}
	if css != v.css {
		v.css = css
		v.needStyle = true
	}
}

// Stats returns the work counters.
func (v *View) Stats() Stats {
	s := v.stats
	s.Render = v.cache.Stats
	return s
}

// Warnings returns the warnings of the current document.
func (v *View) Warnings() []string {
	if v.doc == nil {
		return nil
	}
	return v.doc.Warnings
}

// update brings the document up to date for a w x h viewport.
func (v *View) update(w, h int) {
	for pass := 0; pass < 3; pass++ {
		if v.parsed == nil {
			v.stats.Parses++
			v.parsed = render.Parse(v.html)
			v.rebind()
			v.syncInputs()
			v.needStyle = true
		}
		if v.needStyle {
			v.doc = v.cache.Style(v.parsed, v.css)
			v.doc.Input = v.inputView
			v.needStyle, v.needLayout = false, true
		}
		if v.needLayout || w != v.w || h != v.h {
			v.w, v.h = w, h
			v.doc.Layout(w, h)
			v.needLayout, v.needPlace = false, true
		}
		if v.needPlace {
			v.place()
			v.needPlace, v.needPaint = false, true
			v.hoverStale = true
		}
		// What is under the pointer may have changed: update :hover, and
		// restyle if a rule depends on it.
		if v.hoverStale && v.hasPointer {
			v.hoverStale = false
			v.setHover(v.hitDoc(v.pointer.X, v.pointer.Y))
			if v.needStyle {
				continue
			}
		}
		return
	}
}

// rebind carries interaction state over to a freshly parsed document:
// focus by element key; active and hover are recomputed from input.
func (v *View) rebind() {
	v.hover, v.active, v.down, v.space, v.dragging = nil, nil, nil, nil, nil
	v.pressed = false
	v.focus = nil
	if v.focusKey != "" {
		if n := findKey(v.parsed.Dom.Root, v.focusKey); n != nil {
			v.focus = n
			n.Dyn |= style.StateFocus
		} else {
			v.focusKey = ""
		}
	}
	v.hoverStale = true
}

func (v *View) place() {
	v.doc.Place(func(b *render.Box) geom.Point {
		if b.Style.Overflow != style.OverflowScroll {
			return geom.Point{}
		}
		k := key(b.El)
		o, ok := v.scroll[k]
		if !ok {
			return o
		}
		c := clampScroll(b, o)
		if c != o {
			v.scroll[k] = c
		}
		return c
	})
}

// Frame renders the view into buf (w x h logical pixels at scale).
// Stages whose inputs did not change are skipped; if nothing changed, the
// retained previous frame is copied.
func (v *View) Frame(buf *image.RGBA, w, h int, scale float64) {
	v.stats.Frames++
	v.update(w, h)
	if v.frame == nil || v.frame.Rect != buf.Rect || scale != v.scale {
		v.frame = image.NewRGBA(buf.Rect)
		v.scale = scale
		v.needPaint = true
	}
	if v.needPaint {
		v.doc.Draw(v.frame, w, h, scale, &v.list, &v.raster)
		v.needPaint = false
	} else {
		v.stats.Copies++
	}
	copy(buf.Pix, v.frame.Pix)
}

// Document returns the current document, brought up to date for the last
// frame's viewport (or nil before the first frame).
func (v *View) Document() *render.Document {
	if v.doc != nil {
		v.update(v.w, v.h)
	}
	return v.doc
}

// Bounds returns the border box of the first element with the given id,
// in logical units.
func (v *View) Bounds(id string) (geom.Rect, bool) {
	d := v.Document()
	if d == nil {
		return geom.Rect{}, false
	}
	n := findKey(d.Dom.Root, "#"+id)
	if n == nil {
		return geom.Rect{}, false
	}
	b := d.BoxOf(n)
	if b == nil {
		return geom.Rect{}, false
	}
	return b.Abs, true
}

// hit returns the element under (x, y) in the up-to-date document, or nil.
func (v *View) hit(x, y geom.Unit) *dom.Node {
	if v.Document() == nil {
		return nil
	}
	return v.hitDoc(x, y)
}

// hitDoc hit-tests the current document without updating it.
func (v *View) hitDoc(x, y geom.Unit) *dom.Node {
	if b := v.doc.HitTest(x, y); b != nil {
		return b.El
	}
	return nil
}
