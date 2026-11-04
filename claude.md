# Project: flexui (working name)

A portable desktop UI framework written in Go. UI is described in HTML plus a strict CSS subset (flexbox only). Interaction logic is written in Go. There is no JavaScript and no embedded browser. Rendering is done by our own pure-Go software renderer.

This project is also an experiment: how much of a UI framework can an LLM build from scratch with fast automated feedback. Build everything listed under "From scratch" yourself. Do not pull in a UI toolkit or browser engine.

## Goals

1. Small, self-contained binaries (target under 15 MB including a bundled font).
2. Identical rendering on every platform (same pixels from the same input).
3. Portability: development happens on Linux in a headless cloud environment (no display, no GPU). The code must then run unchanged on macOS and Windows. iOS and Android are out of scope for now but must not be made impossible by design choices.
4. No cgo anywhere. The core is pure Go, and platform window layers must also build with `CGO_ENABLED=0` (macOS via `purego`, Windows via `golang.org/x/sys/windows`). This is what lets us cross-compile and compile-check all targets from Linux.
5. Everything except the window layer must run and be testable headless (render to PNG, no display needed). Headless is the primary target; real windows are thin adapters on top.
6. Determinism: output must be byte-identical across OSes. Bundle fonts (never read system fonts), use only integer or explicitly specified float math in layout and rasterization (avoid platform-dependent behavior such as `math.FMA` differences or map iteration order), use `path` / `io/fs` instead of OS-specific paths, and treat all text files as `\n` line endings.

## Non-goals (for now)

- Full HTML/CSS compatibility. We are "web-like, not the web".
- JavaScript, WebAssembly, networking, media playback, printing.
- GPU rendering (design the draw interface so a GPU backend could be added later).
- Accessibility (but keep the node tree structured so an accessibility tree can be derived later).
- iOS and Android.

## Allowed dependencies

- `golang.org/x/net/html` (HTML parsing)
- `golang.org/x/image` (font and vector rasterizing helpers)
- `github.com/go-text/typesetting` (text shaping and line breaking), once text is introduced
- `github.com/ebitengine/purego` (macOS window layer only, inside `platform/darwin`)
- `golang.org/x/sys/windows` (Windows window layer only, inside `platform/windows`)
- Go standard library

Anything else: ask before adding. Do not add Gio, Fyne, Ebitengine (as a UI layer), GLFW-based toolkits, Skia bindings, or Yoga/Taffy bindings. Writing the flex algorithm ourselves is part of the test.

## From scratch (must be written in this repo)

- DOM/node tree built from parsed HTML
- Style parsing and resolution for the CSS subset
- Flexbox layout engine
- Painting and the software rasterizer
- Event dispatch, hit testing, focus
- Handler binding between HTML and Go
- State/re-render loop

## Architecture

```
/spec                  subset specification (source of truth, see below)
/internal/dom          node tree, HTML -> tree
/internal/style        CSS parsing, selector matching, computed styles
/internal/layout       flexbox engine; input: styled tree + viewport, output: rects
/internal/paint        display list + software rasterizer -> *image.RGBA
/internal/text         font loading, shaping, line breaking, measurement
/*.go                  public package flexui: App, Ctx, handlers, state, re-render loop
/platform              public interface + implementations
  /platform/headless   renders to image, injects synthetic events (for tests); the reference implementation
  /platform/darwin     real macOS window via purego (written in the cloud, run and verified by the user on a Mac)
  /platform/windows    real Windows window via x/sys/windows (later)
  /platform/linux      X11/Wayland window (later, optional)
/examples        small apps used as acceptance tests
/testdata        golden PNGs and layout fixtures
```

Dependency rule: everything under `internal/` and the root `flexui` package must never import anything from `platform/<os>` or any OS-specific package. Backends import `flexui`, never the other way around.

### Platform interface (keep it small)

```go
type Platform interface {
    // Creates a surface of the given size in logical pixels at the given scale.
    Run(cfg Config, app AppHandler) error
}

type AppHandler interface {
    Frame(buf *image.RGBA, w, h int, scale float64) // draw into buf
    Event(e Event)                                   // pointer, key, text, resize, focus
}
```

Plus clipboard and a monotonic clock/timer source. The headless platform implements the same interface and is used by all tests.

### Interaction model (no JS)

- HTML declares handlers by name: `<button onclick="Save">`.
- Go registers handlers: `app.Handle("Save", func(ctx *Ctx) { ... })`.
- Go owns state. After a handler runs, the view is re-rendered from state. Initial approach: render HTML with `html/template` from the state struct, re-parse, diff or rebuild the tree, preserve focus and scroll state by element `id`.
- Optimize later; correctness first.

## The CSS/HTML subset (write this into /spec/SUBSET.md first)

Start with the following and document every decision. If something is not listed it is unsupported, and unsupported properties must produce a clear warning, not silent misbehavior.

- Elements: `div`, `span`, `p`, `h1`-`h3`, `button`, `input` (text), `img`, `ul`/`li`, `a` (as a clickable, no navigation). Unknown elements behave as `div`.
- Every element is a flex container. Default `flex-direction: column`. No normal flow, floats, inline-block, tables, or margin collapsing.
- Layout properties: `display` (`flex`, `none`), `flex-direction`, `flex-wrap`, `flex-grow`, `flex-shrink`, `flex-basis`, `justify-content`, `align-items`, `align-self`, `align-content`, `gap`, `width`, `height`, `min-*`, `max-*`, `padding`, `margin`, `border-width`, `box-sizing`, `position` (`relative`, `absolute`), `top/right/bottom/left`, `overflow` (`visible`, `hidden`, `scroll`), `z-index`.
- Units: `px`, `%`, `rem`, plus bare numbers where relevant. No `calc()` initially.
- Paint properties: `background-color`, `border-color`, `border-radius`, `opacity`, `box-shadow` (single shadow), `color`, `font-family`, `font-size`, `font-weight`, `line-height`, `text-align`, `text-overflow: ellipsis`, `white-space`.
- Selectors: tag, `.class`, `#id`, descendant, child (`>`), `:hover`, `:focus`, `:active`. No specificity beyond a simple fixed order (document it).
- Sources: inline `style=""` and `<style>` blocks.

## Library design (this must be importable by other projects)

The end product is a Go library, not just an app. Other projects will `go get` it and write apps against it.

- Use a real, importable module path in `go.mod` (placeholder `github.com/<user>/flexui`, I will rename it before publishing). Do not use a local-only name.
- Public API lives in the root package `flexui` and a few small public packages (`flexui/platform` for the interface, `flexui/platform/darwin` etc. for backends). Keep this surface small and documented with Go doc comments.
- Move implementation details (`dom`, `style`, `layout`, `paint`, `text`) under `internal/` so they can change freely without breaking users. Only expose what an app author needs.
- Typical app code should look roughly like this, and `examples/` must be written against only the public API:

```go
app := flexui.New(flexui.Options{Title: "Hello", Width: 800, Height: 600})
app.SetView(viewHTML, styleCSS)           // or a func(state) string
app.Handle("Save", func(c *flexui.Ctx) { /* update state */ })
darwin.Run(app)                           // or headless.Render(app, "out.png")
```

- Importing the library must not drag in OS-specific dependencies for other platforms: backends are separate packages, so an app on Linux never compiles purego or the Windows code.
- Examples live in `examples/` with their own `go.mod` (using a `replace` directive to the local library) so they prove the library works as an external consumer would use it.
- Use semantic versioning with `v0.x` tags until the API settles. Keep a `CHANGELOG.md`.
- Include a LICENSE file (ask me which one, default MIT) and record the bundled font's license separately.

## Milestones and acceptance criteria

Work in order. Do not start a milestone until the previous one passes its tests. Commit at each milestone.

**M0. Spec and scaffolding.** `go.mod`, directory layout, `/spec/SUBSET.md`, CI-friendly `go test ./...` that passes with trivial tests, Makefile or script for common tasks.

**M1. Layout engine.** Hand-built styled trees in, rectangles out. Table-driven unit tests covering: row/column, grow/shrink/basis, wrap, justify/align combinations, gap, padding/margin/border, min/max constraints, percentages, absolute positioning. Text and images are leaf nodes with a measure callback (stub with fixed sizes for now). Cross-check expected values against known CSS flexbox behavior; list any deliberate deviations in the spec.

**M2. Software rasterizer.** Draw filled rects, rounded rects, borders, single box-shadow, opacity, clipping to a rect or rounded rect, anti-aliased edges. Output PNG. Golden-image tests with a small pixel-diff tolerance and a `-update` flag to regenerate goldens.

**M3. HTML and styles to pixels.** Parse HTML, parse CSS subset, compute styles, run layout, paint. A single function: `RenderHTML(html string, css string, w, h int) *image.RGBA`. Golden tests for 10+ small documents.

**M4. Text.** Bundle one open-license font (e.g. Inter or Noto Sans, record license in repo). Shaping, line breaking, wrapping inside flex items, ellipsis, text measurement wired into layout. Golden tests.

**M5. Handlers and state.** The Go handler binding and re-render loop, driven entirely by synthetic events in the headless platform. Tests: click a button, state changes, re-rendered output matches golden.

**M6. Interaction.** Hit testing with z-order, hover/active/focus styles, keyboard focus traversal, scroll containers (wheel and drag) with clipping, scroll position preserved across re-renders.

**M7. Portability gate and macOS window.**
- Add a script `scripts/check-portability.sh` that runs `CGO_ENABLED=0 go build ./...` and `go vet` for `linux/amd64`, `linux/arm64`, `darwin/arm64`, `darwin/amd64`, `windows/amd64`. It must pass from the Linux cloud environment and be part of the pre-commit checklist from M0 onward (platform packages can be stubs until this milestone).
- Add a test proving determinism: the golden suite renders the same bytes regardless of GOOS (compare hashes checked into `testdata`).
- Write `platform/darwin` with purego: open a window, present the pixel buffer, handle HiDPI scale, resize, mouse, keyboard, and text input. You cannot run it in the cloud, so keep it small, follow Apple's documented APIs carefully, and write `platform/darwin/README.md` listing what is untested and how I should verify it on my Mac (exact `go run` command and what I should see). Include a runnable example.
- Windows and Linux windows are deferred; do not start them.

**M8. Text input.** Single-line `input` with caret, selection, copy/paste, basic IME composition on macOS.

**M9. Demo app.** `/examples/demo`: a list, a form, and a modal. It must be usable on macOS and produce the same golden images in headless mode.

## How to work

- Read `/spec/SUBSET.md` before changing behavior; update it in the same commit as any behavior change.
- Prefer small packages with narrow APIs. No circular imports.
- Every feature gets tests. For visual features, add a golden image test and open the PNG to check it actually looks right before committing the golden.
- Run `go vet ./...`, `go test ./...`, and (from M7) `scripts/check-portability.sh` before every commit.
- You are running on a headless Linux machine. Never try to open a window there. To "see" output, render to PNG and open the image.
- Never add OS-specific code outside `/platform/<os>`. No build tags in core packages.
- Avoid per-pixel or per-frame allocations in hot paths; reuse buffers.
- Be honest about gaps: keep `/spec/STATUS.md` listing what works, what is stubbed, and known bugs.
- If a requirement is ambiguous, choose the simplest option, record the decision in the spec, and move on. Ask me only about decisions that are expensive to reverse.
- Keep a short `/spec/ARCHITECTURE.md` updated so a fresh session can resume from the repo alone.

## First task

Do M0 now: create the repo scaffolding and write `/spec/SUBSET.md` from the subset above, tightening any ambiguity. Then proceed to M1 and stop after M1's tests pass, summarizing what you built and any spec decisions you made.
