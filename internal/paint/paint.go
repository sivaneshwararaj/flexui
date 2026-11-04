// Package paint defines the drawing interface (Canvas), a display list
// that records drawing commands (List), and a pure-Go software rasterizer
// (Raster) that renders them into an *image.RGBA.
//
// All coordinates are logical geom.Unit (1/64 px). Converting to device
// pixels, snapping and anti-aliasing are the backend's job, so a GPU
// backend can implement Canvas later without changing producers. The
// software rasterizer uses only integer arithmetic, so output is
// byte-identical on every platform. Its pixel-snapping rule is documented
// on Raster and in spec/SUBSET.md §6.
package paint

import "github.com/sivaneshwararaj/flexui/internal/geom"

// Color is a non-premultiplied sRGB color with alpha.
type Color struct{ R, G, B, A uint8 }

// Opaque returns an opaque color.
func Opaque(r, g, b uint8) Color { return Color{r, g, b, 255} }

// Radius is an elliptical corner radius.
type Radius struct{ X, Y geom.Unit }

// Corner indexes RRect.Radii.
const (
	TopLeft = iota
	TopRight
	BottomRight
	BottomLeft
)

// RRect is a rectangle with optional rounded corners.
type RRect struct {
	Rect  geom.Rect
	Radii [4]Radius // TopLeft, TopRight, BottomRight, BottomLeft
}

// Rect returns an RRect without rounded corners.
func Rect(r geom.Rect) RRect { return RRect{Rect: r} }

// Uniform returns an RRect whose corners all have circular radius rad.
func Uniform(r geom.Rect, rad geom.Unit) RRect {
	rr := RRect{Rect: r}
	for i := range rr.Radii {
		rr.Radii[i] = Radius{rad, rad}
	}
	return rr
}

// Shadow is a single outer box shadow, as in CSS `box-shadow`.
// Blur is the CSS blur radius (the Gaussian's standard deviation is half
// of it). Inset shadows are not supported.
type Shadow struct {
	DX, DY, Blur, Spread geom.Unit
	Color                Color
}

// Canvas receives drawing commands. Implementations: *List (records) and
// *Raster (software rendering).
type Canvas interface {
	// FillRRect fills a (rounded) rectangle.
	FillRRect(r RRect, c Color)
	// Border fills the area between r and r inset by widths. Inner corner
	// radii are the outer radii minus the adjacent widths (as in CSS).
	Border(r RRect, widths geom.Edges, c Color)
	// BoxShadow draws an outer shadow of the box r. The shadow is not
	// drawn inside r itself (as in CSS).
	BoxShadow(r RRect, s Shadow)
	// PushClip intersects the clip with r until the matching PopClip.
	PushClip(r RRect)
	PopClip()
	// PushOpacity starts a group that is composited with the given alpha
	// (0-255) at the matching PopOpacity.
	PushOpacity(alpha uint8)
	PopOpacity()
	// Glyphs draws a run of glyphs. The run must not be modified while
	// a recording holds it.
	Glyphs(run *GlyphRun)
}

// OpKind identifies a recorded command.
type OpKind uint8

const (
	OpFillRRect OpKind = iota
	OpBorder
	OpBoxShadow
	OpPushClip
	OpPopClip
	OpPushOpacity
	OpPopOpacity
	OpGlyphs
)

// Op is one recorded command. Only the fields relevant to Kind are set.
type Op struct {
	Kind   OpKind
	RRect  RRect
	Widths geom.Edges
	Shadow Shadow
	Color  Color
	Alpha  uint8
	Run    *GlyphRun
}

// List is a display list: it records Canvas commands so they can be
// replayed into another Canvas, inspected, or (later) diffed.
type List struct {
	Ops []Op
}

// Reset empties the list, keeping its storage.
func (l *List) Reset() { l.Ops = l.Ops[:0] }

// Replay sends every recorded command to c in order.
func (l *List) Replay(c Canvas) {
	for i := range l.Ops {
		op := &l.Ops[i]
		switch op.Kind {
		case OpFillRRect:
			c.FillRRect(op.RRect, op.Color)
		case OpBorder:
			c.Border(op.RRect, op.Widths, op.Color)
		case OpBoxShadow:
			c.BoxShadow(op.RRect, op.Shadow)
		case OpPushClip:
			c.PushClip(op.RRect)
		case OpPopClip:
			c.PopClip()
		case OpPushOpacity:
			c.PushOpacity(op.Alpha)
		case OpPopOpacity:
			c.PopOpacity()
		case OpGlyphs:
			c.Glyphs(op.Run)
		}
	}
}

func (l *List) FillRRect(r RRect, c Color) {
	l.Ops = append(l.Ops, Op{Kind: OpFillRRect, RRect: r, Color: c})
}
func (l *List) Border(r RRect, w geom.Edges, c Color) {
	l.Ops = append(l.Ops, Op{Kind: OpBorder, RRect: r, Widths: w, Color: c})
}
func (l *List) BoxShadow(r RRect, s Shadow) {
	l.Ops = append(l.Ops, Op{Kind: OpBoxShadow, RRect: r, Shadow: s})
}
func (l *List) PushClip(r RRect)        { l.Ops = append(l.Ops, Op{Kind: OpPushClip, RRect: r}) }
func (l *List) PopClip()                { l.Ops = append(l.Ops, Op{Kind: OpPopClip}) }
func (l *List) PushOpacity(alpha uint8) { l.Ops = append(l.Ops, Op{Kind: OpPushOpacity, Alpha: alpha}) }
func (l *List) PopOpacity()             { l.Ops = append(l.Ops, Op{Kind: OpPopOpacity}) }

var (
	_ Canvas = (*List)(nil)
	_ Canvas = (*Raster)(nil)
)

func (l *List) Glyphs(run *GlyphRun) { l.Ops = append(l.Ops, Op{Kind: OpGlyphs, Run: run}) }
