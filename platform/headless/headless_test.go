package headless

import (
	"image"
	"testing"

	"github.com/sivaneshwararaj/flexui/platform"
)

type recorder struct {
	frames int
	events []platform.EventKind
}

func (r *recorder) Frame(buf *image.RGBA, w, h int, scale float64) { r.frames++ }
func (r *recorder) Event(e platform.Event)                         { r.events = append(r.events, e.Kind) }

func TestRunDeliversEventsAndFrames(t *testing.T) {
	var p Platform
	p.Inject(platform.Event{Kind: platform.EventPointerDown})
	p.Inject(platform.Event{Kind: platform.EventResize, Width: 20, Height: 10})
	r := &recorder{}
	if err := p.Run(platform.Config{Width: 10, Height: 10, Scale: 2}, r); err != nil {
		t.Fatal(err)
	}
	if r.frames != 3 || len(r.events) != 2 {
		t.Fatalf("frames=%d events=%v", r.frames, r.events)
	}
	if got := p.Last.Rect.Dx(); got != 40 {
		t.Fatalf("last frame width = %d, want 40", got)
	}
	var _ platform.Platform = &p
	var _ platform.Clipboard = &p
}
