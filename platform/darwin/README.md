# platform/darwin: macOS window (purego, no cgo)

**Status: written and compile-checked only.** This backend was written on
Linux. It builds and passes `go vet` for `darwin/arm64` and
`darwin/amd64` with `CGO_ENABLED=0` (`scripts/check-portability.sh`), but
it has **never been run**. Nothing here is claimed to work until the
stages below pass on a Mac.

## How it works

* `purego` opens AppKit, QuartzCore, CoreFoundation and CoreGraphics and
  sends Objective-C messages (`objc_darwin.go`).
* `Open` creates `NSApplication`, an `NSWindow` (titled, closable,
  resizable) and a content view of a class registered at run time,
  `FlexUIView` (an `NSView` subclass: flipped, first responder, with a
  tracking area for mouse moves and exits).
* Every frame: the app renders into an `*image.RGBA` (premultiplied RGBA,
  sRGB) at the window's `backingScaleFactor`; the bytes are copied into a
  `CFData`, wrapped in a `CGImage` and set as the view layer's contents
  (`contentsScale` = backing scale), inside a `CATransaction` without
  animation.
* Events from `FlexUIView` (`mouseDown/Up/Dragged/Moved/Exited`,
  `scrollWheel`, `keyDown/Up`) become `platform.Event`s; a frame is
  rendered after each one. Window delegate: `windowDidResize:` re-renders,
  `windowWillClose:` sends `EventClose` and exits the process.
* Keys: special keys by virtual key code (arrows, Home/End, Page
  Up/Down, Return, Tab, Delete/Backspace, Forward Delete, Escape,
  Space); other keys by `charactersIgnoringModifiers`. Modifiers from
  `modifierFlags` (Shift, Control, Option = Alt, Command = Meta). Typed
  text is `[event characters]` sent as `EventText` when no Command or
  Control is held.
* Clipboard: `NSPasteboard generalPasteboard`, plain text.

## Verify it, stage by stage

Run from the repository root on a Mac (Go 1.24+, no Xcode or cgo
needed). Each stage prints `EXPECT:` (what you should see) and `RESULT:`
lines (what it checked itself). Add `-debug` to log every AppKit step,
event and frame (with timings) to stderr; please include that log when
something fails.

| Stage | Command | Checks |
|---|---|---|
| 1 | `go run ./platform/darwin/stages/stage1` | purego loads the frameworks; the Objective-C runtime answers; prints the macOS version. No window. |
| 2 | `go run ./platform/darwin/stages/stage2` | A window opens and shows a test pattern drawn without flexui's renderer: RED top-left, GREEN top-right, BLUE bottom-left, WHITE bottom-right, thin black border. Checks orientation (not upside down) and channel order (red is red). Resize redraws. |
| 3 | `go run ./platform/darwin/stages/stage3` | A static flexui page. `RESULT ... identical: true` means the presented frame is byte-identical to a headless render of the same size and scale; it writes `stage3-window.png` and `stage3-headless.png`. Text must look sharp on Retina. |
| 4 | `go run ./platform/darwin/stages/stage4` | Resizing re-lays out the page (boxes re-wrap, the size label updates); moving to a display with another scale prints a new `RESULT` with the scale. |
| 5 | `go run ./platform/darwin/stages/stage5` | Hover highlights, clicks (`RESULT: Inc ...`), wheel/trackpad and scrollbar-thumb scrolling, clicking list items, Tab focus and Enter/Space activation. |
| 6 | `go run ./platform/darwin/stages/stage6` | Typing into inputs, caret and selection keys (Option = word, Command = line), Cmd+A/C/X/V with the system clipboard, `RESULT: change` when leaving a field. |
| 7 | `go run ./platform/darwin/stages/stage7` | The public entry point `darwin.Run(app)` with a complete small app. |

Runnable example: `cd examples && go run ./counter -window`.

Also run the determinism gate on the Mac (both should pass unchanged):

```
go test ./internal/render -run TestHashes
go test ./...
```

## Untested (everything), and specific risks

* **All of it has only been compiled.** In particular:
  * Struct arguments and returns through `objc_msgSend` via purego
    (`NSRect` to `initWithContentRect:...`, `bounds`, `locationInWindow`,
    `convertPoint:fromView:`), especially `stret` returns on amd64.
  * Registering `FlexUIView` / `FlexUIWindowDelegate` with
    `objc.RegisterClass` and the method type encodings purego derives
    (`B@:` for `isFlipped`, `v@:@` for events).
  * Reading `kCGColorSpaceSRGB` through `Dlsym` and dereferencing it.
  * Whether layer contents are drawn top-down as expected with a
    flipped view (stage 2 checks this) and whether `contentsScale` gives
    1:1 device pixels (stage 3 sharpness).
  * Key codes and modifier bits; `charactersIgnoringModifiers` for
    shortcuts with Command (Cmd+A/C/V should arrive as "a"/"c"/"v").
  * Scroll deltas: sign and the line-to-point factor for non-precise
    wheels (16).
  * Window close exits the process with `os.Exit(0)`; there is no
    graceful shutdown.
* **Not implemented**: IME composition and dead keys
  (`NSTextInputClient`; typed text uses `[event characters]`, so dead
  keys and input methods will not compose), caret blinking, cursor shapes
  (no pointer cursor over links), menus (no Quit menu: close the window
  or press Ctrl+C in the terminal), multiple windows, full-screen,
  dark-mode appearance, accessibility, retina-to-non-retina window moves
  are only handled through `viewDidChangeBackingProperties` (untested).
* Performance: every frame is fully re-rendered and copied into a new
  `CGImage`; fine for app-sized windows, not measured on real hardware.
