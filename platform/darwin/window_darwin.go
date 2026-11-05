//go:build darwin

package darwin

import (
	"fmt"
	"image"
	"os"

	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/sivaneshwararaj/flexui/platform"
)

// AppKit constants.
const (
	nsActivationPolicyRegular = 0
	nsWindowStyleTitled       = 1 << 0
	nsWindowStyleClosable     = 1 << 1
	nsWindowStyleMiniaturize  = 1 << 2
	nsWindowStyleResizable    = 1 << 3
	nsBackingStoreBuffered    = 2

	nsTrackingMouseEnteredAndExited = 0x01
	nsTrackingMouseMoved            = 0x02
	nsTrackingActiveInKeyWindow     = 0x20
	nsTrackingInVisibleRect         = 0x200

	kCGImageAlphaPremultipliedLast = 1
)

// Window is one native window showing an AppHandler's frames.
type Window struct {
	app       platform.AppHandler
	win       objc.ID
	view      objc.ID
	nsApp     objc.ID
	buf       *image.RGBA
	w, h      int     // logical size
	scale     float64 // backing scale factor
	frames    int
	redrawing bool
	marked    string // the IME's marked text (NSTextInputClient)
	// OnFrame, if set, is called after each presented frame (stages use
	// it to report what was drawn).
	OnFrame func(img *image.RGBA, w, h int, scale float64)
}

// the current window: AppKit callbacks find it here (one window per
// process).
var current *Window

var classesReady bool

func registerClasses() error {
	if classesReady {
		return nil
	}
	classesReady = true
	evt := func(name string, fn func(w *Window, e objc.ID)) objc.MethodDef {
		return objc.MethodDef{Cmd: sel(name), Fn: func(self objc.ID, _ objc.SEL, e objc.ID) {
			if current != nil {
				fn(current, e)
			}
		}}
	}
	viewClass, err := objc.RegisterClass("FlexUIView", objc.GetClass("NSView"), nil, nil, []objc.MethodDef{
		{Cmd: sel("isFlipped"), Fn: func(objc.ID, objc.SEL) bool { return true }},
		{Cmd: sel("acceptsFirstResponder"), Fn: func(objc.ID, objc.SEL) bool { return true }},
		{Cmd: sel("acceptsFirstMouse:"), Fn: func(objc.ID, objc.SEL, objc.ID) bool { return true }},
		evt("mouseDown:", func(w *Window, e objc.ID) { w.pointer(e, platform.EventPointerDown) }),
		evt("mouseUp:", func(w *Window, e objc.ID) { w.pointer(e, platform.EventPointerUp) }),
		evt("mouseDragged:", func(w *Window, e objc.ID) { w.pointer(e, platform.EventPointerMove) }),
		evt("mouseMoved:", func(w *Window, e objc.ID) { w.pointer(e, platform.EventPointerMove) }),
		evt("mouseExited:", func(w *Window, e objc.ID) { w.send(platform.Event{Kind: platform.EventPointerLeave}) }),
		evt("scrollWheel:", func(w *Window, e objc.ID) { w.wheel(e) }),
		evt("keyDown:", func(w *Window, e objc.ID) { w.key(e, true) }),
		evt("keyUp:", func(w *Window, e objc.ID) { w.key(e, false) }),
		{Cmd: sel("viewDidChangeBackingProperties"), Fn: func(objc.ID, objc.SEL) {
			if current != nil {
				debugf("viewDidChangeBackingProperties")
				current.render()
			}
		}},
	})
	if err != nil {
		return fmt.Errorf("darwin: FlexUIView: %w", err)
	}
	if err := addTextInputClient(viewClass); err != nil {
		return err
	}
	_, err = objc.RegisterClass("FlexUIWindowDelegate", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("windowDidResize:"), Fn: func(objc.ID, objc.SEL, objc.ID) {
			if current != nil {
				debugf("windowDidResize")
				current.render()
			}
		}},
		{Cmd: sel("windowWillClose:"), Fn: func(objc.ID, objc.SEL, objc.ID) {
			debugf("windowWillClose: exiting")
			if current != nil {
				current.app.Event(platform.Event{Kind: platform.EventClose})
			}
			os.Exit(0)
		}},
	})
	if err != nil {
		return fmt.Errorf("darwin: FlexUIWindowDelegate: %w", err)
	}
	return nil
}

// Open creates the application and a window for app, without running the
// event loop. The window is shown and its first frame rendered.
func Open(cfg platform.Config, app platform.AppHandler) (*Window, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	if err := registerClasses(); err != nil {
		return nil, err
	}
	w := &Window{app: app}
	current = w
	w.nsApp = class("NSApplication").Send(sel("sharedApplication"))
	w.nsApp.Send(sel("setActivationPolicy:"), nsActivationPolicyRegular)
	debugf("NSApp %#x", w.nsApp)

	rect := NSRect{Size: NSSize{float64(cfg.Width), float64(cfg.Height)}}
	style := nsWindowStyleTitled | nsWindowStyleClosable | nsWindowStyleMiniaturize | nsWindowStyleResizable
	w.win = class("NSWindow").Send(sel("alloc")).Send(sel("initWithContentRect:styleMask:backing:defer:"),
		rect, uint64(style), uint64(nsBackingStoreBuffered), false)
	if w.win == 0 {
		return nil, fmt.Errorf("darwin: NSWindow init failed")
	}
	w.win.Send(sel("setReleasedWhenClosed:"), false)
	w.win.Send(sel("setTitle:"), nsString(cfg.Title))
	w.win.Send(sel("setAcceptsMouseMovedEvents:"), true)
	w.win.Send(sel("setDelegate:"), class("FlexUIWindowDelegate").Send(sel("new")))

	w.view = class("FlexUIView").Send(sel("alloc")).Send(sel("initWithFrame:"), rect)
	w.view.Send(sel("setWantsLayer:"), true)
	area := class("NSTrackingArea").Send(sel("alloc")).Send(sel("initWithRect:options:owner:userInfo:"),
		NSRect{}, uint64(nsTrackingMouseEnteredAndExited|nsTrackingMouseMoved|nsTrackingActiveInKeyWindow|nsTrackingInVisibleRect),
		w.view, objc.ID(0))
	w.view.Send(sel("addTrackingArea:"), area)
	w.win.Send(sel("setContentView:"), w.view)
	w.win.Send(sel("makeFirstResponder:"), w.view)
	w.win.Send(sel("center"))
	w.win.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
	w.nsApp.Send(sel("activateIgnoringOtherApps:"), true)
	debugf("window %#x view %#x layer %#x", w.win, w.view, w.view.Send(sel("layer")))
	w.render()
	return w, nil
}

// Loop runs the AppKit event loop. It returns only if the loop is
// stopped; closing the window exits the process.
func (w *Window) Loop() {
	debugf("[NSApp run]")
	w.nsApp.Send(sel("run"))
}

// Size reports the window's content size in points and its backing
// scale factor.
func (w *Window) Size() (width, height int, scale float64) {
	b := objc.Send[NSRect](w.view, sel("bounds"))
	scale = objc.Send[float64](w.win, sel("backingScaleFactor"))
	if scale <= 0 {
		scale = 1
	}
	return int(b.Size.Width + 0.5), int(b.Size.Height + 0.5), scale
}

// render asks the app for a frame at the current size and scale and
// presents it.
func (w *Window) render() {
	t0 := time.Now()
	lw, lh, scale := w.Size()
	if lw <= 0 || lh <= 0 {
		return
	}
	if lw != w.w || lh != w.h {
		if w.w != 0 {
			w.app.Event(platform.Event{Kind: platform.EventResize, Width: lw, Height: lh})
		}
		w.w, w.h = lw, lh
	}
	w.scale = scale
	pw, ph := int(float64(lw)*scale+0.5), int(float64(lh)*scale+0.5)
	if w.buf == nil || w.buf.Rect.Dx() != pw || w.buf.Rect.Dy() != ph {
		w.buf = image.NewRGBA(image.Rect(0, 0, pw, ph))
	}
	w.app.Frame(w.buf, lw, lh, scale)
	w.present()
	w.frames++
	debugf("frame %d: %dx%d pt @%gx (%dx%d px) in %v", w.frames, lw, lh, scale, pw, ph, time.Since(t0))
	if w.OnFrame != nil {
		w.OnFrame(w.buf, lw, lh, scale)
	}
}

// present sets the frame as the view layer's contents: the pixels are
// copied into a CFData, wrapped as a CGImage (sRGB, premultiplied RGBA).
func (w *Window) present() {
	img := w.buf
	data := cfDataCreate(0, &img.Pix[0], len(img.Pix))
	provider := cgDataProviderCreateWithCFData(data)
	space := cgColorSpaceCreateWithName(kCGColorSpaceSRGB)
	cg := cgImageCreate(img.Rect.Dx(), img.Rect.Dy(), 8, 32, img.Stride, space,
		kCGImageAlphaPremultipliedLast, provider, 0, false, 0)
	layer := w.view.Send(sel("layer"))
	class("CATransaction").Send(sel("begin"))
	class("CATransaction").Send(sel("setDisableActions:"), true)
	layer.Send(sel("setContentsScale:"), w.scale)
	layer.Send(sel("setContents:"), objc.ID(cg))
	class("CATransaction").Send(sel("commit"))
	cfRelease(cg)
	cfRelease(space)
	cfRelease(provider)
	cfRelease(data)
}

func (w *Window) send(e platform.Event) {
	debugf("event %+v", e)
	w.app.Event(e)
	w.render()
}

// location returns the event position in view coordinates (points,
// top-left origin since the view is flipped).
func (w *Window) location(e objc.ID) NSPoint {
	p := objc.Send[NSPoint](e, sel("locationInWindow"))
	return objc.Send[NSPoint](w.view, sel("convertPoint:fromView:"), p, objc.ID(0))
}

func (w *Window) pointer(e objc.ID, kind platform.EventKind) {
	p := w.location(e)
	w.send(platform.Event{Kind: kind, X: p.X, Y: p.Y, Button: 0})
}

func (w *Window) wheel(e objc.ID) {
	p := w.location(e)
	dx := objc.Send[float64](e, sel("scrollingDeltaX"))
	dy := objc.Send[float64](e, sel("scrollingDeltaY"))
	if !objc.Send[bool](e, sel("hasPreciseScrollingDeltas")) {
		dx, dy = dx*16, dy*16 // lines to points
	}
	// AppKit deltas move the content; flexui's move the viewport.
	w.send(platform.Event{Kind: platform.EventWheel, X: p.X, Y: p.Y, DX: -dx, DY: -dy})
}

func (w *Window) key(e objc.ID, down bool) {
	code := objc.Send[uint16](e, sel("keyCode"))
	m := mods(objc.Send[uint64](e, sel("modifierFlags")))
	chars := goString(e.Send(sel("charactersIgnoringModifiers")))
	name, special := keyNames[code]
	if !special {
		name = chars
	}
	kind := platform.EventKeyUp
	if down {
		kind = platform.EventKeyDown
	}
	if down && w.marked != "" {
		// During an IME composition the input method gets every key
		// (arrows, Enter and Escape act on the marked text).
		debugf("key during composition: interpretKeyEvents")
		w.view.Send(sel("interpretKeyEvents:"), class("NSArray").Send(sel("arrayWithObject:"), e))
		w.render()
		return
	}
	w.app.Event(platform.Event{Kind: kind, Key: name, Mods: m})
	debugf("key %v code %#x name %q mods %#x", kind, code, name, m)
	if down && m&(platform.ModCtrl|platform.ModMeta) == 0 {
		// Text goes through the input system: insertText: for committed
		// text (also plain typing and dead keys), setMarkedText: for an
		// IME composition (ime_darwin.go). Commands such as deleteBackward:
		// are ignored (the key event above handled them).
		w.view.Send(sel("interpretKeyEvents:"), class("NSArray").Send(sel("arrayWithObject:"), e))
	}
	w.render()
}

// App is what Run needs (*flexui.App satisfies it).
type App interface {
	platform.AppHandler
	Config() platform.Config
}

type clipboardUser interface{ SetClipboard(platform.Clipboard) }

// Run opens a window for app (its Config gives the title and size), gives
// it the system clipboard, and runs the event loop. Call it from the main
// goroutine. Closing the window exits the process.
func Run(app App) error {
	return Platform{}.Run(app.Config(), app)
}

// Platform implements platform.Platform.
type Platform struct{}

// Run implements platform.Platform.
func (Platform) Run(cfg platform.Config, app platform.AppHandler) error {
	if cu, ok := app.(clipboardUser); ok {
		cu.SetClipboard(Clipboard{})
	}
	w, err := Open(cfg, app)
	if err != nil {
		return err
	}
	w.Loop()
	return nil
}

// LastFrame returns the last presented frame and its logical size and
// scale.
func (w *Window) LastFrame() (*image.RGBA, int, int, float64) { return w.buf, w.w, w.h, w.scale }

// Redraw renders and presents a new frame now (e.g. after state changed
// outside an event). It is a no-op when called from within OnFrame of a
// frame of the same size.
func (w *Window) Redraw() {
	if w.redrawing {
		return
	}
	w.redrawing = true
	defer func() { w.redrawing = false }()
	w.render()
}
