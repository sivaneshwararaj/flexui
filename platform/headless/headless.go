// Package headless is the reference platform: it renders frames into an
// in-memory image and lets tests inject synthetic events. It never opens a
// window and works without a display or GPU.
package headless

import (
	"image"

	"github.com/sivaneshwararaj/flexui/platform"
)

// Platform is a headless platform. Events queued with Inject are delivered
// in order during Run, and a frame is rendered after the initial setup and
// after every event.
type Platform struct {
	Events []platform.Event
	// Last holds the most recently rendered frame after Run returns.
	Last *image.RGBA
	clip string
}

// Inject queues an event for delivery during Run.
func (p *Platform) Inject(e platform.Event) { p.Events = append(p.Events, e) }

// Run implements platform.Platform.
func (p *Platform) Run(cfg platform.Config, app platform.AppHandler) error {
	scale := cfg.Scale
	if scale == 0 {
		scale = 1
	}
	w, h := cfg.Width, cfg.Height
	frame := func() {
		pw, ph := int(float64(w)*scale), int(float64(h)*scale)
		if p.Last == nil || p.Last.Rect.Dx() != pw || p.Last.Rect.Dy() != ph {
			p.Last = image.NewRGBA(image.Rect(0, 0, pw, ph))
		}
		app.Frame(p.Last, w, h, scale)
	}
	frame()
	for _, e := range p.Events {
		if e.Kind == platform.EventResize {
			w, h = e.Width, e.Height
		}
		app.Event(e)
		frame()
	}
	p.Events = p.Events[:0]
	return nil
}

// ReadText implements platform.Clipboard with an in-memory clipboard.
func (p *Platform) ReadText() (string, error) { return p.clip, nil }

// WriteText implements platform.Clipboard with an in-memory clipboard.
func (p *Platform) WriteText(s string) error { p.clip = s; return nil }
