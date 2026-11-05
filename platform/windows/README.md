# platform/windows: Windows window (x/sys/windows, no cgo)

**Status: partly verified on Windows.** This backend was written on Linux
and compile-checked there for `windows/amd64` and `windows/arm64` with
`CGO_ENABLED=0`. The user ran the first stages on Windows (build 26300):

| Stage | Result |
|---|---|
| 1 | Pass: all Win32 functions resolved; `Windows 10.0 build 26300`. |
| 2 | Pass: quadrants red/green/blue/white in the right places (top-down DIB and BGRA conversion correct). The client area in the screenshot is about 600x400 pixels for 480x320 logical pixels, consistent with 125% scaling (DPI sizing works). |
| 3 | Pass: `presented 420x260 @1.25x ... identical: true` (presented frame byte-identical to headless); card, bold and link runs sharp at 125%. |
| 4 | Pass: boxes re-wrap on resize, size text updates; RESULT lines report e.g. `535x320 logical @1.25x = 669x400 pixels`. Moving to a monitor with another scale not reported. |
| 5 | Pass: hover, `Inc` clicks, list wheel scrolling with scrollbar, item selection (`picked item 3`, `5`). |
| 6 | Partial: typing, caret and `RESULT: change` lines work. **Ctrl+C did not copy.** Paste from other apps not reported. The stage now prints a `RESULT: clipboard write/read ...` line with the error for every copy and paste, and `OpenClipboard` is retried for ~100 ms; to be re-run. |
| 7 | Not run yet. |

Everything else (stages 3-7: flexui frames, resizing, DPI changes,
input, text, clipboard, closing) is still untested.

## How it works

* `golang.org/x/sys/windows` (v0.41.0) loads `user32`, `gdi32` and
  `kernel32` lazily and calls them (`win32_windows.go`); stage 1 checks
  that every function resolves.
* `Open` makes the process per-monitor DPI aware (v2), registers the
  window class `FlexUIWindow` with a Go window procedure
  (`windows.NewCallback`), creates an overlapped window and sizes its
  client area to the configured logical size at the window's DPI
  (`AdjustWindowRectExForDpi`). The scale is `GetDpiForWindow / 96`
  (1.25, 1.5, ... are fine: flexui's scale is in 1/64 steps).
* Frames: after each message that matters, the app renders into an
  `*image.RGBA` at the device size; it is converted to BGRA and drawn by
  `SetDIBitsToDevice` (top-down DIB) in `WM_PAINT`. `WM_ERASEBKGND` is
  suppressed to avoid flicker. `WM_SIZE` and `WM_DPICHANGED` (which moves
  the window to the suggested rectangle) re-render.
* Input: `WM_MOUSEMOVE`, left button down/up (with mouse capture, so drags
  outside the window keep working), `WM_MOUSELEAVE` (via
  `TrackMouseEvent`), `WM_MOUSEWHEEL`/`WM_MOUSEHWHEEL` (screen to client
  coordinates; one notch = 48 logical px, the default 3 lines),
  `WM_KEYDOWN/UP` (virtual keys: arrows, Home/End, Page Up/Down,
  Backspace, Delete, Enter, Tab, Escape, Space, letters as lower-case
  names for shortcuts, digits), modifiers from `GetKeyState` (Shift,
  Ctrl, Alt; the Windows key is not mapped), typed text from `WM_CHAR`
  (UTF-16, surrogate pairs joined, control characters skipped).
  `WM_SYSKEYDOWN` is also passed to Windows so Alt+F4 works.
* Shortcuts follow Windows conventions through flexui's editing model:
  Ctrl+Arrow moves by word, Home/End to the line start/end,
  Ctrl+A/C/X/V.
* Clipboard: `CF_UNICODETEXT` through `OpenClipboard`/`GlobalAlloc`.
* Closing the window destroys it, sends `EventClose` and posts `WM_QUIT`;
  `Run` then returns normally (unlike macOS, which exits the process).

## Verify it, stage by stage

On Windows 10 (1703 or later) or 11, from the repository root (Go 1.24+,
no C compiler needed):

| Stage | Command | Checks |
|---|---|---|
| 1 | `go run ./platform/windows/stages/stage1` | All Win32 functions resolve; prints the Windows version. No window. |
| 2 | `go run ./platform/windows/stages/stage2` | A window shows a test pattern drawn without flexui's renderer: RED top-left, GREEN top-right, BLUE bottom-left, WHITE bottom-right, thin black border. Checks orientation (top-down DIB) and channel order (BGRA conversion). Resize redraws; closing prints `message loop ended`. |
| 3 | `go run ./platform/windows/stages/stage3` | A static flexui page; `identical: true` means the presented frame is byte-identical to a headless render at the same size and scale. Writes `stage3-window.png`, `stage3-headless.png`. Sharp at 125%/150% scaling. |
| 4 | `go run ./platform/windows/stages/stage4` | Resizing re-lays out; moving to a monitor with another scaling (or changing it in Settings > Display) prints a new `RESULT` with the scale. |
| 5 | `go run ./platform/windows/stages/stage5` | Hover, clicks (`RESULT: Inc ...`), wheel and scrollbar-thumb scrolling, item selection, Tab focus, Enter/Space. |
| 6 | `go run ./platform/windows/stages/stage6` | Typing, caret keys (Ctrl = word, Home/End = line), Shift selection, Ctrl+A/C/X/V with the Windows clipboard, an emoji typed with Win+. inserts as one character, `RESULT: change` when leaving a field. |
| 7 | `go run ./platform/windows/stages/stage7` | `windows.Run(app)` with a complete app; closing returns normally. |

Each stage prints `EXPECT:` and `RESULT:` lines; add `-debug` to log every
Win32 step, message and frame (with timings) to stderr, and include that
log when something fails. Runnable examples: `cd examples && go run
./counter -window` and `go run ./demo -window`. Also run the determinism
gate there: `go test ./internal/render -run TestHashes` and `go test ./...`.

The expected images for stages 2 and 3 are the same as macOS's:
`review/M7/stage2_expected.png`, `review/M7/stage3_expected*.png` (copied
into `review/windows`).

## Untested parts and specific risks

Stages 1-5 confirm the Win32 bindings, window creation, presentation,
DPI sizing at 125%, resizing, pointer, wheel and keyboard input; stage 6
confirms text input. Ctrl+C (clipboard write) failed in the first run and
is being diagnosed. Still unverified, with the risks noted before running:

* In particular:
  * The hand-written structure layouts (`WNDCLASSEXW`, `MSG`,
    `PAINTSTRUCT`, `BITMAPINFOHEADER`, `TRACKMOUSEEVENT`) and
    `DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2` as the handle value -4.
  * The window procedure as a `windows.NewCallback` and messages arriving
    during `CreateWindowExW` (they go to `DefWindowProcW` until the window
    handle is known).
  * `SetDIBitsToDevice` with a negative height (top-down) and the BGRA
    conversion (stage 2 checks both).
  * Client size and DPI: logical size = client pixels / scale, rounded;
    at fractional scales the frame can be one device pixel smaller than
    the client area (the rest is not painted).
  * `WM_DPICHANGED` handling (the suggested rectangle), and
    `SetProcessDpiAwarenessContext` failing on older Windows (logged; the
    window would then be scaled by Windows).
  * Wheel sign and step, high-resolution touchpad wheels (deltas smaller
    than 120 are passed through proportionally).
  * Key mapping on non-US layouts (letters use virtual-key codes for
    shortcuts; text comes from `WM_CHAR`), AltGr (reported as Ctrl+Alt by
    Windows; text should still arrive through `WM_CHAR`).
  * Clipboard ownership and `GlobalFree` on failure.
* **Not implemented**: IME composition (`WM_IME_*`; input methods may
  still commit text through `WM_CHAR`, which is untested), caret blinking,
  cursor shapes (always the arrow), menus, multiple windows, full screen,
  dark title bar, accessibility, the Windows key as a modifier, touch and
  pen input, high-contrast mode.
* Performance: each frame is fully converted to BGRA and drawn; flexui's
  damage regions are not used for partial `InvalidateRect` yet.
