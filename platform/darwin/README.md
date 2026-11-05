# platform/darwin: macOS window (purego, no cgo)

**Status: partly verified on a Mac.** This backend was written on Linux
and compile-checked there for `darwin/arm64` and `darwin/amd64` with
`CGO_ENABLED=0`. The user ran the stages on a MacBook Air (Apple
silicon, macOS 27.2, build 26B123, Retina, scale 2):

| Stage | Result |
|---|---|
| 1 | Pass: frameworks load, NSApplication found, version printed, no window. |
| 2 | Pass: window opens; quadrants red/green/blue/white in the right places (orientation and channel order correct). Colors look shifted on screen because the frame is tagged sRGB and macOS converts it to the display's color space (expected). |
| 3 | Pass: page sharp at 2x; `identical: true` (presented frame byte-identical to the headless render). |
| 4 | Pass: `460x320 @2x`, boxes wrap three per row; live resizing re-lays out correctly (reported, together with stage 6). No difference noticed in scaling; a move to a display with another scale was not reported. |
| 5 | Pass: clicks change the count, list items select, the list scrolls with the scrollbar shown, a focus border moves to the buttons. Hover highlighting was not visible in the screenshot (pointer elsewhere). |
| 6 | Pass for typing: text appears in both fields, the caret moves, "Echo" mirrors the first field, and leaving it printed `RESULT: change name = "see"`; the window resized cleanly. Clipboard (Cmd+C/V across apps) and Option/Cmd caret moves were not explicitly reported. |
| 7 | Pass: `darwin.Run(app)`; the count reached 80 with the note shown. |
| 8 | Not run yet (multilingual clipboard round trip). |
| 9 | Not run yet (IME composition). |

Everything not in a "Pass" line above (the system clipboard, Option/Cmd
caret movement, moves between displays of different scale, amd64 Macs)
is still untested.

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
| 8 | `go run ./platform/darwin/stages/stage8` | Multilingual clipboard: 13 strings (Tamil, Hindi, Arabic, Hebrew, mixed, combining marks, emoji, joiners) written to and read back from the system clipboard, byte-identical (UTF-8 via `NSString` (`public.utf8-plain-text`)); the Tamil sample left on the clipboard pastes intact into TextEdit; text copied in another app and pasted into the field is printed as code points. Missing Tamil glyphs in the window are expected until the font work lands. |
| 9 | `go run ./platform/darwin/stages/stage9` | IME composition through NSTextInputClient (`ime_darwin.go`): typing goes through `interpretKeyEvents:`; `setMarkedText:selectedRange:replacementRange:` → `EventCompose`, `insertText:replacementRange:` → commit (or plain text), `unmarkText` → commit; `markedRange`, `selectedRange` and `firstRectForCharacterRange:actualRange:` return structs, which purego callbacks cannot, so they are Go assembly (`ime_darwin_arm64.s`, `ime_darwin_amd64.s`) returning values the Go side keeps updated. RESULT lines for every compose/commit/cancel and input event, plus the caret rectangle. |

Runnable examples: `cd examples && go run ./counter -window`, and the demo
app (list, form, modal): `cd examples && go run ./demo -window`. The demo's
headless goldens are in `examples/demo/testdata`; the window should look
the same at scale 1 (and the same layout, sharper, at 2x).

Also run the determinism gate on the Mac (both should pass unchanged):

```
go test ./internal/render -run TestHashes
go test ./...
```

## Untested parts and specific risks

### Stage 9 (IME) risks, all untested

* The assembly method implementations: register use and struct-return
  conventions (arm64: X0/X1 and F0-F3; x86_64: AX/DX and a hidden result
  pointer in DI for NSRect) are written from the ABI documents, not run.
  A mistake here crashes the process when the input method asks for the
  marked range or the caret rectangle.
* `objc.GetProtocol("NSTextInputClient")` may return nil (then the view
  does not declare the protocol and AppKit may not create an input
  context; `-debug` logs it).
* NSRange arguments taken as two uint64 (fine for arm64 and x86_64
  calling conventions; `replacementRange` is ignored).
* `interpretKeyEvents:` now handles all typing (previously characters were
  read from the key event): dead keys and Option-characters should work
  better, but plain typing is untested on a Mac after this change.
* `attributedSubstringForProposedRange:` returns nil and
  `characterIndexForPoint:` NSNotFound: some input methods (e.g. Chinese
  with reconversion) may want the document's text.

Stages 1-5 and 7 confirm window creation, presentation, Retina scale,
pointer clicks, scrolling and keyboard focus on Apple silicon. Still
unverified:

* The system clipboard and Option/Cmd caret keys (stage 6), backing-scale
  changes between displays, Intel (amd64) Macs. Risks noted before
  running:
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
