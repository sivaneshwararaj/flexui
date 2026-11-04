// Package headless is the reference platform: it renders frames into an
// in-memory image and lets tests inject synthetic events. It never opens a
// window and works without a display or GPU.
package headless

import (
	"image"
	"image/png"
	"os"

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

// ClipboardUser is implemented by apps that use a clipboard
// (*flexui.App); platforms hand them theirs.
type ClipboardUser interface {
	SetClipboard(platform.Clipboard)
}

// Run implements platform.Platform.
func (p *Platform) Run(cfg platform.Config, app platform.AppHandler) error {
	if cu, ok := app.(ClipboardUser); ok {
		cu.SetClipboard(p)
	}
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

// App is what Render needs: something that draws frames and knows its
// preferred configuration (*flexui.App satisfies it).
type App interface {
	platform.AppHandler
	Config() platform.Config
}

// RenderImage renders one frame of app at its configured size.
func RenderImage(app App) (*image.RGBA, error) {
	var p Platform
	if err := p.Run(app.Config(), app); err != nil {
		return nil, err
	}
	return p.Last, nil
}

// Render renders one frame of app and writes it to path as a PNG.
func Render(app App, path string) error {
	img, err := RenderImage(app)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Session drives an app step by step for tests: every input method
// delivers one event and renders a frame, like a real platform's loop.
type Session struct {
	app   App
	w, h  int
	scale float64
	frame *image.RGBA
	// Clipboard is the session's in-memory clipboard, given to the app.
	Clipboard Platform
}

// Start renders the first frame of app at its configured size.
func Start(app App) *Session {
	cfg := app.Config()
	s := &Session{app: app, w: cfg.Width, h: cfg.Height, scale: cfg.Scale}
	if s.scale == 0 {
		s.scale = 1
	}
	if cu, ok := app.(ClipboardUser); ok {
		cu.SetClipboard(&s.Clipboard)
	}
	s.render()
	return s
}

func (s *Session) render() {
	pw, ph := int(float64(s.w)*s.scale+0.5), int(float64(s.h)*s.scale+0.5)
	if s.frame == nil || s.frame.Rect.Dx() != pw || s.frame.Rect.Dy() != ph {
		s.frame = image.NewRGBA(image.Rect(0, 0, pw, ph))
	}
	s.app.Frame(s.frame, s.w, s.h, s.scale)
}

// Send delivers one event and renders a frame.
func (s *Session) Send(e platform.Event) {
	if e.Kind == platform.EventResize {
		s.w, s.h = e.Width, e.Height
	}
	s.app.Event(e)
	s.render()
}

// Move moves the pointer to (x, y) in logical pixels.
func (s *Session) Move(x, y float64) {
	s.Send(platform.Event{Kind: platform.EventPointerMove, X: x, Y: y})
}

// Leave moves the pointer off the surface.
func (s *Session) Leave() { s.Send(platform.Event{Kind: platform.EventPointerLeave}) }

// Drag presses at (x0, y0), moves to (x1, y1) in steps, and releases.
func (s *Session) Drag(x0, y0, x1, y1 float64, steps int) {
	s.Move(x0, y0)
	s.Press(x0, y0)
	for i := 1; i <= steps; i++ {
		s.Move(x0+(x1-x0)*float64(i)/float64(steps), y0+(y1-y0)*float64(i)/float64(steps))
	}
	s.Release(x1, y1)
}

// Press presses the primary button at (x, y).
func (s *Session) Press(x, y float64) {
	s.Send(platform.Event{Kind: platform.EventPointerDown, X: x, Y: y})
}

// Release releases the primary button at (x, y).
func (s *Session) Release(x, y float64) {
	s.Send(platform.Event{Kind: platform.EventPointerUp, X: x, Y: y})
}

// Click moves to (x, y), presses and releases there.
func (s *Session) Click(x, y float64) {
	s.Move(x, y)
	s.Press(x, y)
	s.Release(x, y)
}

// Wheel scrolls by (dx, dy) logical pixels at (x, y).
func (s *Session) Wheel(x, y, dx, dy float64) {
	s.Send(platform.Event{Kind: platform.EventWheel, X: x, Y: y, DX: dx, DY: dy})
}

// Key presses and releases a key ("Tab", "Enter", " ", ...) with the given
// modifiers.
func (s *Session) Key(key string, mods uint8) {
	s.Send(platform.Event{Kind: platform.EventKeyDown, Key: key, Mods: mods})
	s.Send(platform.Event{Kind: platform.EventKeyUp, Key: key, Mods: mods})
}

// Type sends committed text, as a keyboard or input method does for
// printable characters.
func (s *Session) Type(text string) {
	s.Send(platform.Event{Kind: platform.EventText, Text: text})
}

// Resize changes the surface size.
func (s *Session) Resize(w, h int) {
	s.Send(platform.Event{Kind: platform.EventResize, Width: w, Height: h})
}

// Frame returns the latest frame. It is overwritten by the next event.
func (s *Session) Frame() *image.RGBA { return s.frame }

// Save writes the latest frame to path as a PNG.
func (s *Session) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, s.frame); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
