# flexui

A portable desktop UI library for Go. Views are **HTML plus a strict,
flexbox-only CSS subset**; logic and state are **Go**. Rendering is done by
flexui's own pure-Go software renderer: no browser engine, no JavaScript,
no cgo, and the same input produces the same pixels on every OS and
architecture.

```go
app := flexui.New(flexui.Options{Title: "Hello", Width: 800, Height: 600})
app.SetViewFunc(view, css)                       // view() returns HTML from your state
app.Handle("Save", func(c *flexui.Ctx) { ... })  // <button onclick="Save">
darwin.Run(app)  // macOS window;  windows.Run(app) on Windows;  headless for tests
```

* **Using flexui (people and coding agents): read [docs/GUIDE.md](docs/GUIDE.md).**
  It covers the model, the whole public API, events, inputs, the HTML/CSS
  subset, testing and pitfalls. Agents: see also [AGENTS.md](AGENTS.md).
* Exact behavior: [spec/SUBSET.md](spec/SUBSET.md). What works and what
  does not yet: [spec/STATUS.md](spec/STATUS.md). Internals:
  [spec/ARCHITECTURE.md](spec/ARCHITECTURE.md).
* Example apps (public API only): [examples/](examples) — `hello`,
  `counter`, `demo` (a contacts manager with a list, a form and a
  modal) and [`showcase`](examples/showcase) (a tour: multi-select order
  form with live totals, calculator, six languages, layout gallery).
  `cd examples && go run ./showcase -window` opens it on macOS or
  Windows; without `-window` it writes PNGs of a scripted session.

## Install

```
go get github.com/sivaneshwararaj/flexui
```

Go 1.24+. Pure Go: cross-compiles with `CGO_ENABLED=0` for Linux, macOS
and Windows (amd64, arm64).

## Features

* Flexbox layout engine (CSS Flexbox §9), verified against Chromium on
  2,500 layout cases; text layout (bundled Inter font, shaping, wrapping,
  inline elements, ellipsis) verified against Chromium to 1/64 px;
  script itemization and font fallback with opt-in Noto font packs for
  Tamil and Devanagari ([fonts](fonts)), also checked against Chromium.
* Painting: backgrounds, borders, rounded corners, box shadows, opacity,
  z-index stacking, clipping, images (PNG/JPEG), anti-aliased text,
  HiDPI.
* Interaction: click/key/focus/input/change handlers bound by name,
  hover/active/focus/disabled styles, keyboard focus traversal, scroll
  containers with scrollbars, single-line text inputs with selection and
  clipboard.
* Re-render on every state change, with incremental work (unchanged
  text is not re-shaped; only damaged screen rectangles are repainted).
* Headless platform for tests: synthetic input, byte-exact golden images.

## Platforms

| Platform | Package | Status |
|---|---|---|
| Headless (tests, PNG) | `platform/headless` | complete |
| macOS | `platform/darwin` | works (verified on Apple silicon) |
| Windows | `platform/windows` | first stages verified; see its README |
| Linux window | — | not yet |

## Status

Pre-1.0 (`v0.x`): the API may change. Not supported yet: IME
composition, menus, multiple windows, grid/tables/floats, italics, CSS
animations. See [spec/STATUS.md](spec/STATUS.md) and the
[CHANGELOG](CHANGELOG.md).

## Developing flexui itself

`make check` runs gofmt, vet, all tests (both modules) and the
portability gate. `make crosscheck` regenerates the Chromium reference
data (needs node and Playwright). The project's working rules are in
[claude.md](claude.md).

## License

MIT ([LICENSE](LICENSE)). The bundled Inter font and the optional Noto
font packs ([fonts](fonts)) are under the SIL Open Font License. The
JPEG decoder in [internal/jpeg](internal/jpeg) is copied from Go 1.24
and keeps Go's BSD license ([internal/jpeg/LICENSE](internal/jpeg/LICENSE)).
