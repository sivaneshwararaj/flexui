// Package view keeps a rendered document alive between frames: it
// re-renders only the stages that changed (parse, style, layout, place,
// paint), does hit testing, and turns pointer and key input into DOM-like
// events (click, ...) dispatched to named handlers (spec/SUBSET.md §8).
//
// A View is not safe for concurrent use; the app runtime serializes
// frames and events.
package view

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"os"

	"github.com/sivaneshwararaj/flexui/internal/dom"
	"github.com/sivaneshwararaj/flexui/internal/geom"
	"github.com/sivaneshwararaj/flexui/internal/paint"
	"github.com/sivaneshwararaj/flexui/internal/render"
	"github.com/sivaneshwararaj/flexui/internal/style"
)

// Stats counts the work done, for tests and tuning.
type Stats struct {
	Restyles    int   // incremental (paint-only) restyles
	Damaged     int   // frames repainted only in damaged rectangles
	DamagePixel int64 // device pixels repainted in those frames
	Frames      int   // Frame calls
	Parses      int   // HTML parses
	Copies      int   // frames served from the retained image without painting
	Render      render.Stats
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

	frame   *image.RGBA // retained output of the last paint
	lists   [2]paint.List
	cur     int  // lists[cur] is the display list of frame
	hasList bool // lists[cur] is valid for frame
	raster  paint.Raster
	scratch *image.RGBA // FLEXUI_CHECK_DAMAGE: full repaint for comparison
	check   bool
	// The platform buffer the last frame was copied into and the device
	// rectangles painted since (nil: all), so the same buffer only
	// gets the changed pixels.
	lastBuf *image.RGBA
	dirty   []image.Rectangle
	full    bool

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
	restyle              []*dom.Node           // elements whose pseudo-class state changed
	scroll               map[string]geom.Point // scroll offsets by element key
	inputs               map[string]*inputState
	selecting            *dom.Node // input whose text is being selected by dragging
	clip                 Clipboard

	stats Stats
}

// New returns an empty view.
func New() *View {
	return &View{cache: render.NewCache(), needStyle: true, scroll: map[string]geom.Point{},
		inputs: map[string]*inputState{}, clip: &memClipboard{},
		check: os.Getenv("FLEXUI_CHECK_DAMAGE") != ""}
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
		if !v.needStyle && len(v.restyle) > 0 {
			// Incremental restyle: paint-only changes keep the layout.
			if v.doc.Restyle(v.restyle) {
				v.needPaint = true
				v.stats.Restyles++
				if v.check {
					v.verifyRestyle()
				}
			} else {
				v.needStyle = true
			}
		}
		v.restyle = v.restyle[:0]
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
			if v.needStyle || len(v.restyle) > 0 {
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
	v.restyle = v.restyle[:0]
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

func (v *View) place() { v.placeDoc(v.doc) }

func (v *View) placeDoc(d *render.Document) {
	d.Place(func(b *render.Box) geom.Point {
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
		v.hasList = false
	}
	if v.needPaint {
		v.paint(w, h)
		v.needPaint = false
	} else {
		v.stats.Copies++
	}
	// Copy to the platform's buffer: only what changed when it is the
	// same buffer as last time (platforms must not modify it in between).
	if buf != v.lastBuf || v.full {
		copy(buf.Pix, v.frame.Pix)
	} else {
		for _, r := range v.dirty {
			for y := r.Min.Y; y < r.Max.Y; y++ {
				i := buf.PixOffset(r.Min.X, y)
				copy(buf.Pix[i:i+4*r.Dx()], v.frame.Pix[i:i+4*r.Dx()])
			}
		}
	}
	v.lastBuf, v.dirty, v.full = buf, v.dirty[:0], false
}

// paint repaints the retained frame (spec/SUBSET.md §6.6): the new display
// list is compared with the previous one, and only the damaged device
// rectangles are repainted, unless the structure changed or most of the
// frame is damaged.
func (v *View) paint(w, h int) {
	s64 := paint.Scale64(v.scale)
	next := 1 - v.cur
	nl := &v.lists[next]
	v.doc.Record(nl, w, h)
	var rects []image.Rectangle
	ok := false
	if v.hasList {
		rects, ok = v.raster.Damage(&v.lists[v.cur], nl, s64, v.frame.Rect)
	}
	var area int64
	for _, r := range rects {
		area += int64(r.Dx()) * int64(r.Dy())
	}
	frameArea := int64(v.frame.Rect.Dx()) * int64(v.frame.Rect.Dy())
	switch {
	case ok && len(rects) == 0:
		// Nothing visible changed.
	case ok && area*10 < frameArea*6:
		for _, r := range rects {
			v.raster.BeginClip(v.frame, s64, r)
			nl.Replay(&v.raster)
			v.raster.End()
		}
		v.dirty = append(v.dirty, rects...)
		v.stats.Damaged++
		v.stats.DamagePixel += area
		if v.check {
			v.verify(nl, s64)
		}
	default:
		v.raster.Begin(v.frame, s64)
		nl.Replay(&v.raster)
		v.raster.End()
		v.full = true
	}
	v.cur, v.hasList = next, true
}

// verifyRestyle (FLEXUI_CHECK_DAMAGE=1) rebuilds and lays out the
// document from scratch and panics if its display list differs from the
// incrementally restyled one.
func (v *View) verifyRestyle() {
	saved := v.cache.Stats
	defer func() { v.cache.Stats = saved }()
	full := v.cache.Style(v.parsed, v.css)
	full.Input = v.inputView
	full.Layout(v.w, v.h)
	v.placeDoc(full)
	var a, b paint.List
	full.Record(&a, v.w, v.h)
	v.doc.Record(&b, v.w, v.h)
	if !paint.SameList(&a, &b) {
		panic("view: incremental restyle differs from a full restyle")
	}
}

// verify (FLEXUI_CHECK_DAMAGE=1) repaints the whole frame and panics if
// it differs from the incrementally repainted one.
func (v *View) verify(l *paint.List, s64 int64) {
	if v.scratch == nil || v.scratch.Rect != v.frame.Rect {
		v.scratch = image.NewRGBA(v.frame.Rect)
	}
	var r paint.Raster
	r.Begin(v.scratch, s64)
	l.Replay(&r)
	r.End()
	if !bytes.Equal(v.scratch.Pix, v.frame.Pix) {
		n := 0
		for i := range v.frame.Pix {
			if v.frame.Pix[i] != v.scratch.Pix[i] {
				n++
			}
		}
		panic(fmt.Sprintf("view: damage repaint differs from a full repaint in %d bytes", n))
	}
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
