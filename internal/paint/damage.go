package paint

import (
	"image"
	"slices"

	"github.com/sivaneshwararaj/flexui/internal/geom"
)

// Damage-region repainting (spec/SUBSET.md §6.6).
//
// Every raster operation is pixel-local under a clip: the value of a
// pixel depends only on the operations that touch it, in order (shadow
// masks are computed over their whole extent; only compositing is
// clipped). So if two display lists have the same structure and differ
// only in some operations, the pixels outside the device bounds of those
// operations are identical, and replaying the new list with a base clip
// equal to a damaged rectangle reproduces exactly the full repaint inside
// it.

// BeginClip is Begin with the base clip restricted to clip (device
// pixels): only pixels inside it are written.
func (r *Raster) BeginClip(dst *image.RGBA, scale64 int64, clip image.Rectangle) {
	r.Begin(dst, scale64)
	r.clips[0].bounds = dst.Rect.Intersect(clip)
}

// opBounds returns the device pixels an operation can touch (before
// clipping), at the raster's current scale.
func (r *Raster) opBounds(op *Op) image.Rectangle {
	switch op.Kind {
	case OpFillRRect, OpBorder, OpPushClip, OpImage:
		s := r.shapeOf(op.RRect)
		return s.bounds()
	case OpBoxShadow:
		if op.Shadow.Color.A == 0 {
			return image.Rectangle{}
		}
		_, _, mb, ok := r.shadowGeometry(op.RRect, op.Shadow)
		if !ok {
			return image.Rectangle{}
		}
		return mb
	case OpGlyphs:
		run := op.Run
		if run.Color.A == 0 {
			return image.Rectangle{}
		}
		ppem := geom.Unit(r.dev(run.Size))
		var u image.Rectangle
		for _, g := range run.Glyphs {
			q := (r.dev(g.X) + 8) >> 4
			ox, sub := int(q>>2), int(q&3)
			oy := snap(r.dev(g.Y))
			m := run.Font.GlyphMask(g.ID, ppem, sub)
			if m == nil || m.W == 0 {
				continue
			}
			u = u.Union(image.Rect(ox+m.X, oy+m.Y, ox+m.X+m.W, oy+m.Y+m.H))
		}
		return u
	}
	return image.Rectangle{}
}

func sameRun(a, b *GlyphRun) bool {
	return a == b || a != nil && b != nil && a.Font == b.Font && a.Size == b.Size &&
		a.Color == b.Color && slices.Equal(a.Glyphs, b.Glyphs)
}

func sameOp(a, b *Op) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == OpGlyphs {
		return sameRun(a.Run, b.Run)
	}
	return *a == *b
}

// Damage compares the display lists of two consecutive frames painted at
// scale64 into frame and returns the device rectangles where their pixels
// can differ (merged, at most 8). ok is false when the lists differ in
// structure (length or kind of operation): repaint everything then.
func (r *Raster) Damage(prev, next *List, scale64 int64, frame image.Rectangle) (rects []image.Rectangle, ok bool) {
	r.scale = scale64
	if len(prev.Ops) != len(next.Ops) || !sameKinds(prev.Ops, next.Ops) {
		return r.damageSpliced(prev.Ops, next.Ops, frame)
	}
	type group struct {
		changed    bool
		accA, accB image.Rectangle
	}
	clipA := []image.Rectangle{frame}
	clipB := []image.Rectangle{frame}
	var groups []group
	var dmg []image.Rectangle
	add := func(rc image.Rectangle) {
		if !rc.Empty() {
			dmg = append(dmg, rc)
		}
	}
	for i := range next.Ops {
		a, b := &prev.Ops[i], &next.Ops[i]
		if a.Kind != b.Kind {
			return nil, false
		}
		ca, cb := clipA[len(clipA)-1], clipB[len(clipB)-1]
		switch a.Kind {
		case OpPushClip:
			ba, bb := r.opBounds(a).Intersect(ca), r.opBounds(b).Intersect(cb)
			if !sameOp(a, b) {
				add(ba.Union(bb))
			}
			clipA, clipB = append(clipA, ba), append(clipB, bb)
			continue
		case OpPopClip:
			if len(clipA) > 1 {
				clipA, clipB = clipA[:len(clipA)-1], clipB[:len(clipB)-1]
			}
			continue
		case OpPushOpacity:
			groups = append(groups, group{changed: a.Alpha != b.Alpha})
			continue
		case OpPopOpacity:
			if n := len(groups); n > 0 {
				g := groups[n-1]
				groups = groups[:n-1]
				if g.changed {
					add(g.accA.Union(g.accB))
				}
			}
			continue
		}
		same := sameOp(a, b)
		if same && len(groups) == 0 {
			continue // bounds only matter for changes and groups
		}
		ba, bb := r.opBounds(a).Intersect(ca), r.opBounds(b).Intersect(cb)
		for k := range groups {
			groups[k].accA = groups[k].accA.Union(ba)
			groups[k].accB = groups[k].accB.Union(bb)
		}
		if !same {
			add(ba.Union(bb))
		}
	}
	return mergeRects(dmg, frame), true
}

// mergeRects merges overlapping or touching rectangles, and everything
// into one bounding box if more than 8 remain.
func mergeRects(rs []image.Rectangle, frame image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for _, rc := range rs {
		rc = rc.Intersect(frame)
		if rc.Empty() {
			continue
		}
		for merged := true; merged; {
			merged = false
			for k := 0; k < len(out); k++ {
				if out[k].Inset(-1).Overlaps(rc) {
					rc = rc.Union(out[k])
					out = append(out[:k], out[k+1:]...)
					merged = true
					break
				}
			}
		}
		out = append(out, rc)
	}
	if len(out) > 8 {
		var u image.Rectangle
		for _, rc := range out {
			u = u.Union(rc)
		}
		out = []image.Rectangle{u}
	}
	// Deterministic order.
	slices.SortFunc(out, func(a, b image.Rectangle) int {
		if a.Min.Y != b.Min.Y {
			return a.Min.Y - b.Min.Y
		}
		return a.Min.X - b.Min.X
	})
	return out
}

func sameKinds(a, b []Op) bool {
	for i := range a {
		if a[i].Kind != b[i].Kind {
			return false
		}
	}
	return true
}

// paintState is the clip and opacity-group state while walking a list.
type paintState struct {
	clips  []image.Rectangle
	alphas []uint8
}

func (st *paintState) equal(o *paintState) bool {
	return slices.Equal(st.clips, o.clips) && slices.Equal(st.alphas, o.alphas)
}

// step applies op to the state and returns the clipped device bounds it
// paints (empty for clip and group operations).
func (r *Raster) step(st *paintState, op *Op) image.Rectangle {
	c := st.clips[len(st.clips)-1]
	switch op.Kind {
	case OpPushClip:
		st.clips = append(st.clips, r.opBounds(op).Intersect(c))
	case OpPopClip:
		if len(st.clips) > 1 {
			st.clips = st.clips[:len(st.clips)-1]
		}
	case OpPushOpacity:
		st.alphas = append(st.alphas, op.Alpha)
	case OpPopOpacity:
		if len(st.alphas) > 0 {
			st.alphas = st.alphas[:len(st.alphas)-1]
		}
	default:
		return r.opBounds(op).Intersect(c)
	}
	return image.Rectangle{}
}

// damageSpliced handles lists of different structure (an operation added
// or removed, e.g. a background that appears on :hover): the common
// prefix and suffix are identical, so only the two middle parts can
// differ. If both middles leave the clip and group state the same, the
// suffix paints the same pixels, and the damage is what the middles
// paint. Otherwise the whole frame is damaged (ok false).
func (r *Raster) damageSpliced(a, b []Op, frame image.Rectangle) ([]image.Rectangle, bool) {
	p := 0
	for p < len(a) && p < len(b) && sameOp(&a[p], &b[p]) {
		p++
	}
	q := 0
	for q < len(a)-p && q < len(b)-p && sameOp(&a[len(a)-1-q], &b[len(b)-1-q]) {
		q++
	}
	st := paintState{clips: []image.Rectangle{frame}}
	for i := 0; i < p; i++ {
		r.step(&st, &a[i])
	}
	var dmg []image.Rectangle
	walk := func(ops []Op) paintState {
		s := paintState{clips: slices.Clone(st.clips), alphas: slices.Clone(st.alphas)}
		for i := range ops {
			if rc := r.step(&s, &ops[i]); !rc.Empty() {
				dmg = append(dmg, rc)
			}
		}
		return s
	}
	ea := walk(a[p : len(a)-q])
	eb := walk(b[p : len(b)-q])
	if !ea.equal(&eb) {
		return nil, false
	}
	return mergeRects(dmg, frame), true
}

// SameList reports whether two display lists are identical (for checks).
func SameList(a, b *List) bool {
	if len(a.Ops) != len(b.Ops) {
		return false
	}
	for i := range a.Ops {
		if !sameOp(&a.Ops[i], &b.Ops[i]) {
			return false
		}
	}
	return true
}
