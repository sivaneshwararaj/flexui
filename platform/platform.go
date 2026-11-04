// Package platform defines the small interface between the core (which is
// pure Go and OS-independent) and a window system. Implementations live in
// subpackages: headless (reference, used by tests), darwin, windows, linux.
//
// The core packages (dom, style, layout, paint, text, app) must never import
// this package's subpackages.
package platform

import "image"

// Config describes the surface a platform should create.
type Config struct {
	Title  string
	Width  int     // logical pixels
	Height int     // logical pixels
	Scale  float64 // device pixels per logical pixel; 0 means "platform default"
}

// EventKind identifies the type of an Event.
type EventKind uint8

const (
	EventNone EventKind = iota
	EventPointerMove
	EventPointerDown
	EventPointerUp
	EventWheel
	EventKeyDown
	EventKeyUp
	EventText
	EventResize
	EventFocus
	EventBlur
	EventClose
	EventPointerLeave // the pointer left the surface
)

// Event is a platform-independent input event. Coordinates are in logical
// pixels. The exact field set will grow in M5/M6/M8.
type Event struct {
	Kind   EventKind
	X, Y   float64 // pointer position
	DX, DY float64 // wheel delta in logical pixels; positive DY scrolls down
	Button int
	Key    string // key name for key events: "Tab", "Enter", " ", "Escape", "a", ...
	Text   string // committed text for EventText
	Mods   uint8  // ModShift | ModCtrl | ModAlt | ModMeta
	Width  int    // new size for EventResize
	Height int
}

// Modifier key bits for Event.Mods.
const (
	ModShift uint8 = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// AppHandler is implemented by the application runtime (package app).
type AppHandler interface {
	// Frame draws the current UI into buf, which is w*h logical pixels
	// multiplied by scale.
	Frame(buf *image.RGBA, w, h int, scale float64)
	// Event delivers one input event.
	Event(e Event)
}

// Platform runs an AppHandler on some kind of surface.
type Platform interface {
	Run(cfg Config, app AppHandler) error
}

// Clipboard is optionally implemented by platforms.
type Clipboard interface {
	ReadText() (string, error)
	WriteText(s string) error
}
