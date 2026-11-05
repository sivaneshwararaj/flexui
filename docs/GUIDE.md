# flexui guide (for developers and coding agents)

This is the complete guide to building an app with flexui as a library.
It is written to be read top to bottom once, then used as a reference.
Everything here is checked against the code; the snippets marked
"(tested)" are Go examples in `example_test.go` that `go test` runs.

* Spec (source of truth for exact behavior): [`spec/SUBSET.md`](../spec/SUBSET.md)
* Current state and known gaps: [`spec/STATUS.md`](../spec/STATUS.md)
* Package reference: `go doc github.com/sivaneshwararaj/flexui` (and
  `.../platform`, `.../platform/headless`, `.../platform/darwin`,
  `.../platform/windows`)

## 1. Mental model

* A flexui app is a Go program. The UI is **HTML plus a strict CSS
  subset**, rendered by flexui's own pure-Go renderer (no browser, no
  JavaScript, no cgo). The same input gives the same pixels on every OS.
* **Go owns the state.** The view is a function of the state that returns
  HTML (usually an `html/template`). HTML elements name Go handlers
  (`onclick="Save"`); handlers change the state; flexui calls the view
  function again and renders the new HTML. There is no virtual DOM to
  manage and no JS: re-rendering is cheap and automatic.
* **Layout is flexbox only.** Every element is a flex container, column by
  default. There is no block flow, float, grid or table layout.
* An app is an `*flexui.App`. A **platform** runs it: `platform/headless`
  (in memory, for tests and PNG output), `platform/darwin` (macOS window),
  `platform/windows` (Windows window).

## 2. Install and import

```
go get github.com/sivaneshwararaj/flexui
```

Go 1.24 or later. No C compiler. Imports:

```go
import (
	"github.com/sivaneshwararaj/flexui"                    // App, Ctx, RenderHTML, CheckHTML
	"github.com/sivaneshwararaj/flexui/platform/headless" // tests, PNGs
	"github.com/sivaneshwararaj/flexui/platform/darwin"   // macOS window (only builds on darwin)
	"github.com/sivaneshwararaj/flexui/platform/windows"  // Windows window (only builds on windows)
)
```

The window backends only compile on their OS. Put the call in files with
build tags, as `examples/counter` does (`window_darwin.go`,
`window_windows.go`, `window_other.go`).

## 3. The smallest programs

Render one page to an image (tested: `ExampleRenderHTML`):

```go
img := flexui.RenderHTML(`<div style="width: 50px; height: 20px; background-color: #3478f6"></div>`, "", 100, 40)
// img is an *image.RGBA of 100x40 device pixels (scale 1).
warnings := flexui.CheckHTML(html, css) // unsupported elements, properties, selectors
```

A stateful app driven headlessly (tested: `ExampleApp`):

```go
type state struct{ Count int }
st := &state{}
tmpl := template.Must(template.New("view").Parse(
	`<div class="row"><button id="inc" onclick="Inc">+</button><div id="n">{{.Count}}</div></div>`))

app := flexui.New(flexui.Options{Title: "Counter", Width: 200, Height: 80})
app.SetViewFunc(func() string {
	var b bytes.Buffer
	tmpl.Execute(&b, st)
	return b.String()
}, `.row { flex-direction: row; gap: 8px; padding: 8px; align-items: center; }`)
app.Handle("Inc", func(c *flexui.Ctx) { st.Count++ })

s := headless.Start(app)          // first frame
x, y, w, h, _ := app.Bounds("inc") // find an element by id
s.Click(x+w/2, y+h/2)               // press + release; Inc runs; view re-renders
```

Open it in a window instead (in an OS-specific file):

```go
if err := darwin.Run(app); err != nil { log.Fatal(err) }   // macOS
if err := windows.Run(app); err != nil { log.Fatal(err) }  // Windows
```

`Run` must be called from the main goroutine. It blocks until the window
closes (macOS exits the process on close; Windows returns).

## 4. The App API

| Call | Meaning |
|---|---|
| `flexui.New(flexui.Options{Title, Width, Height, Scale, Assets})` | Create an app. Width/Height in logical pixels (default 800x600). Scale 0 = platform decides. `Assets` is an `fs.FS` for `<img src>` paths (e.g. an `embed.FS`). |
| `app.SetView(html, css)` | Static view. |
| `app.SetViewFunc(func() string, css)` | View generated from state. Called again before the next frame after any handler ran or after `Invalidate`. Returning identical HTML costs nothing. |
| `app.Handle(name, func(*flexui.Ctx))` | Bind a handler name used in HTML attributes. |
| `app.Invalidate()` | State changed outside a handler (timer, goroutine): re-run the view. Safe from any goroutine. |
| `app.Bounds(id) (x, y, w, h, ok)` | Border box of the element with that id, in logical pixels (as of the last frame). For inline elements, the union of their fragments. |
| `app.Value(id)` | Current text of an `<input>`. |
| `app.Focus(id) bool`, `app.FocusedID()` | Focus an element ("" blurs), read focus. |
| `app.ScrollOffset(id) (x, y)` | Scroll position of a scroll container. |
| `app.Warnings()` | Warnings of the current view (unsupported HTML/CSS) plus handler names with no registered handler. **Check this in tests.** |
| `app.SetClipboard(platform.Clipboard)` | Platforms do this; default is an in-memory clipboard. |
| `app.Config()`, `app.Frame(...)`, `app.Event(...)` | The `platform.AppHandler` interface used by backends. |
| `flexui.RenderHTML(html, css, w, h)` | One-shot render at scale 1 on a white canvas. |
| `flexui.CheckHTML(html, css)` | Warnings only. |

**Threading.** Frames, events and handlers run on one goroutine (the
platform's UI loop). Inside a handler, **do not call `App` methods that
take the app's lock** (`Focus`, `Value`, `Bounds`, `Warnings`,
`ScrollOffset`, `SetView...`, `Handle`): they deadlock. Use the `Ctx`
instead (`c.Value`, `c.InputValue`, `c.Attr`, `c.Focus`). `Invalidate` is always safe.

## 5. Handlers and events

Declare handlers in HTML with these attributes (others warn):

| Attribute | Fires | Bubbles |
|---|---|---|
| `onclick` | primary-button press + release (on the nearest common ancestor of both), Enter on a focused `button`/`a`, Space released on a focused `button` | yes |
| `onkeydown` | key pressed; target is the focused element, or `body` | yes |
| `onfocus`, `onblur` | focus changes (pointer, Tab, `Focus`) | no |
| `oninput` | the text of an `<input>` changed (typing, delete, cut, paste) | yes |
| `onchange` | an `<input>` lost focus or got Enter, if its value changed | yes |

Bubbling: the target's handler runs first, then each ancestor's, until
one calls `c.StopPropagation()`. A `disabled` `button`/`input` receives no
clicks, and neither do its ancestors through it.

The `*flexui.Ctx` a handler receives:

| Field / method | Meaning |
|---|---|
| `c.Event` | `"click"`, `"keydown"`, `"focus"`, `"blur"`, `"input"`, `"change"` |
| `c.ID` | id of the element whose attribute is running |
| `c.TargetID` | id of the deepest element the event is for (may be a descendant) |
| `c.Attr(name)` | any attribute of the element whose handler runs, e.g. `data-id` |
| `c.Value` | text of the input (input, change, keydown in an input) |
| `c.InputValue(id)` | text of any `<input>` by id (handler-safe `app.Value`) |
| `c.Key`, `c.Mods` | key name and modifiers (`platform.ModShift`, `ModCtrl`, `ModAlt`, `ModMeta`) |
| `c.X`, `c.Y` | pointer position (pointer events), logical pixels |
| `c.StopPropagation()` | stop bubbling |
| `c.PreventDefault()` | for keydown: suppress Tab navigation and Enter/Space activation (and input editing) |
| `c.Focus(id)` | focus an element **after** the re-render (it may be one the new view adds) |
| `c.App()` | the app (do not call locking methods on it inside the handler) |

Key names: `"Enter"`, `"Tab"`, `" "` (Space), `"Escape"`, `"Backspace"`,
`"Delete"`, `"ArrowLeft"`, `"ArrowRight"`, `"ArrowUp"`, `"ArrowDown"`,
`"Home"`, `"End"`, `"PageUp"`, `"PageDown"`, and letters/digits as
lower-case characters (`"a"`, `"7"`). Typed text does not come as key
events but as text (see inputs).

**Lists**: put the item identity in a data attribute and read it with
`c.Attr`:

```html
{{range .Items}}<div class="row" onclick="Select" data-id="{{.ID}}">{{.Name}}</div>{{end}}
```

```go
app.Handle("Select", func(c *flexui.Ctx) { st.Selected = c.Attr("data-id") })
```

**Modal dialogs**: render the overlay only when open, as a positioned
element with a high `z-index`; close on Escape with an `onkeydown` on an
outer element, and on a backdrop click by checking `c.TargetID`:

```html
{{if .Confirm}}<div id="backdrop" class="backdrop" onclick="Backdrop">
  <div class="dialog">...<button onclick="Cancel">Cancel</button></div></div>{{end}}
```

```css
.backdrop { position: absolute; left: 0; top: 0; right: 0; bottom: 0; z-index: 10;
  background-color: rgba(15, 23, 42, 0.45); align-items: center; justify-content: center; }
```

```go
app.Handle("Backdrop", func(c *flexui.Ctx) { if c.TargetID == "backdrop" { st.Confirm = false } })
```

`examples/demo` is a complete app (list, search, form, modal) to copy
from.

## 6. State and re-rendering

* After any handler runs, the view function is called before the next
  frame. The new HTML replaces the document (no diffing needed in your
  code). Work is skipped where the input did not change: identical HTML
  is not re-parsed; unchanged text is not re-shaped; only changed screen
  rectangles are repainted.
* Interaction state survives re-renders, matched by element **`id`** (or
  by tree position for elements without one): focus, scroll positions,
  and the text/caret of inputs. **Give stable ids** to inputs, scroll
  containers and anything you focus.
* Escape user data in templates (`html/template` does it for you).
* The HTML string is parsed with a full HTML5 parser; you can return a
  fragment (no `<html>`/`<body>` needed). `<style>` elements in the HTML
  are applied after the `css` argument.

## 7. Text inputs

`<input>` is a single-line text field. Two patterns work:

* **Controlled**: `value="{{.Name}}"` plus `oninput` storing `c.Value` in
  the state (tested: `ExampleCtx_value`). The caret stays where it is
  while typing.
* **Uncontrolled**: no handler; in a handler read the text with
  `c.InputValue(id)` (tested: `ExampleCtx_InputValue`), outside handlers
  with `app.Value(id)`.

Rules: the `value` attribute sets the text when the input first appears
and whenever the attribute **changes** between renders; set the state to a
new value to replace the text programmatically. `placeholder` shows when
empty. `disabled` makes it read-only and unfocusable. Editing keys:
arrows (Alt/Ctrl = word, Meta = line), Home/End, Backspace/Delete,
Ctrl/Meta+A/C/X/V; the pointer places the caret and drag-selects.
In headless tests type with `s.Type("text")` (committed text) and use
`s.Key(name, mods)` for editing keys.

## 8. HTML elements

| Element | Behavior |
|---|---|
| `div`, `p`, `h1`-`h3`, `ul`, `li` | flex container (no default margins, bullets or fonts) |
| `span`, `strong`, `b`, `em`, `i`, `a`, `br` | inline (text flows across them); `strong`/`b` bold; `em`/`i` upright (no italic font); `a` blue, clickable and focusable, `href` ignored |
| `button` | flex row, centered, default border/background/hover/active/focus/disabled styles |
| `input` | text field, 160px wide, one line tall, default styles |
| `img` | PNG or JPEG from `src`: a `data:` URL or a path in `Options.Assets`; natural size 1 image px = 1 logical px; keeps aspect ratio |
| `body` | the root; fills the window |
| anything else | behaves as `div` and warns |

Default styles are in `internal/render/ua.css`; your CSS always wins over
them.

## 9. CSS subset (cheat sheet)

Defaults differ from browsers: **every element is `display: flex`,
`flex-direction: column`, `position: relative`**; borders are solid
whenever they have a width; `box-sizing` defaults to `content-box`.
Units: `px`, `%`, `rem` (always 16px), `0`. No `calc()`, `em`, `vw`.

Layout: `display` (`flex`, `inline`, `none`), `position` (`relative`,
`absolute`), `top/right/bottom/left`, `flex-direction`, `flex-wrap`,
`flex`, `flex-grow`, `flex-shrink`, `flex-basis`, `justify-content`,
`align-items`, `align-self`, `align-content`, `gap`, `row-gap`,
`column-gap`, `width`, `height`, `min-*`, `max-*`, `margin` (incl.
`auto`), `padding`, `border-width`, `box-sizing`, `overflow` (`visible`,
`hidden`, `scroll` — one value for both axes), `z-index`.

Paint: `background-color`/`background` (a color only), `border`,
`border-color` (one color), `border-style` (`solid`, `none`),
`border-radius` (no `/`), `opacity`, `box-shadow` (one outer shadow),
`color`. Colors: hex, `rgb()`/`rgba()`, named, `transparent`,
`currentcolor`.

Text: `font-size`, `font-weight` (> 500 is bold), `line-height`,
`text-align` (`left`, `right`, `center`), `white-space` (`normal`,
`nowrap`, `pre`, `pre-wrap`), `text-overflow: ellipsis` (with
`overflow: hidden` and usually `white-space: nowrap`). `font-family` is
accepted but every family renders with the bundled Inter.

**Other scripts.** Inter covers Latin, Greek and Cyrillic. For other
scripts import a font pack once, anywhere in your program (it registers
itself; order does not matter):

```go
import (
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"      // Noto Sans Tamil, ~95 KB
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari" // Noto Sans Devanagari (Hindi, Marathi, Nepali), ~365 KB
)
```

Text is split by script and each cluster Inter lacks is drawn with the
first pack that has it, as Chromium does. Without a pack, such text is
drawn as boxes (the text itself is kept intact). Put `lang="mr"` etc.
on an element for language-specific letter forms. Right-to-left
scripts (Arabic, Hebrew) are not supported yet. See
[fonts/README.md](../fonts/README.md).

Selectors: type, `.class`, `#id`, `*`, compound, descendant (`a b`),
child (`a > b`), lists (`a, b`), `:hover`, `:focus`, `:active`,
`:disabled`. Specificity works as in CSS; inline `style=""` wins; no
`!important`.

Not supported (they warn): grid, floats, tables, `display: block`,
`position: fixed/sticky`, transforms, transitions/animations, gradients,
`hsl()`, attribute selectors, `:nth-child`, pseudo-elements, media
queries, `@font-face`, `vertical-align`, `letter-spacing`, italics, and
padding/border/margin on inline elements.

**Common layout recipes**

```css
.row    { flex-direction: row; align-items: center; gap: 8px; }  /* horizontal stack */
.spacer { flex-grow: 1; }                                        /* push the rest right */
.fill   { flex-grow: 1; flex-basis: 0; }                         /* share space equally */
.list   { overflow: scroll; flex-grow: 1; flex-basis: 0; }       /* scrolling region */
.list > .item { flex-shrink: 0; }                                /* items keep their height */
.center { align-items: center; justify-content: center; }
.title  { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.overlay { position: absolute; left: 0; top: 0; right: 0; bottom: 0; z-index: 10; }
```

Gotcha: in a scroll container, children shrink to fit by default (flex
column); give them `flex-shrink: 0`, otherwise nothing overflows and
nothing scrolls.

## 10. Images and assets

```go
//go:embed assets
var assets embed.FS
app := flexui.New(flexui.Options{Width: 800, Height: 600, Assets: assets})
// <img src="assets/logo.png" style="width: 32px">
```

`data:image/png;base64,...` URLs work without `Assets`. Remote URLs are
not loaded (no networking). Unloadable images warn and leave an empty box.

## 11. Testing apps headlessly

`headless.Session` steps an app one event and one frame at a time:

| Method | Does |
|---|---|
| `headless.Start(app)` | render the first frame at `app.Config()` size |
| `s.Click(x, y)`, `s.Press`, `s.Release`, `s.Move`, `s.Leave`, `s.Drag(x0, y0, x1, y1, steps)` | pointer |
| `s.Wheel(x, y, dx, dy)` | scroll (logical px; positive dy scrolls down) |
| `s.Key(name, mods)` | key down + up |
| `s.Type(text)` | committed text into the focused input |
| `s.Resize(w, h)`, `s.Send(platform.Event{...})` | other events |
| `s.Frame()`, `s.Save(path)` | the latest frame (`*image.RGBA`), as PNG |
| `s.Clipboard` | the session's in-memory clipboard |

Find elements with `app.Bounds(id)`. Assert on your state, on
`app.Warnings()` (should be empty), and on frames: rendering is
deterministic and integer-only, so a golden PNG can be compared **byte
for byte** on every OS and architecture (see `examples/demo/demo_test.go`
for a scripted session with goldens and an `-update` flag).
`headless.Render(app, "out.png")` writes a single frame.

Setting `FLEXUI_CHECK_DAMAGE=1` in tests makes flexui verify every
incremental repaint against a full one (slower; catches renderer bugs).

## 12. Platforms

| Package | Status |
|---|---|
| `platform/headless` | reference platform, used by all tests |
| `platform/darwin` | macOS window via purego; verified on Apple silicon (stages 1-7) |
| `platform/windows` | Windows window via x/sys/windows; partly verified (stages 1-2) |
| Linux window | not implemented (headless works on Linux) |

Each backend's README lists what is verified and untested, and has
staged commands to check it on a real machine. Not yet supported on any
window backend: IME composition, menus, multiple windows, cursor shapes.

## 13. Pitfalls checklist

1. Forgot `flex-direction: row`? Everything stacks vertically by default.
2. Nothing scrolls: the container needs `overflow: scroll` and a bounded
   size (`height`, or `flex-grow: 1; flex-basis: 0` in a bounded parent),
   and its children `flex-shrink: 0`.
3. A handler never runs: check `app.Warnings()` for "no handler
   registered" and for unsupported `on*` attributes.
4. A deadlock in a handler: you called an `App` method; use `Ctx`.
5. Focus or input text lost after re-render: give the element a stable
   `id`.
6. Text in a link or span ignores `padding`/`margin`: inline elements
   take no box properties; wrap in a `div` (a flex item) instead.
7. CSS silently "not applied": run `flexui.CheckHTML` and read the
   warnings; unsupported properties are reported, never half-applied.
8. Layout differs from a browser: see the numbered deviations in
   `spec/SUBSET.md` §4, §6, §7.5 (e.g. column default, position relative
   default, no italics, scale-independent layout).
