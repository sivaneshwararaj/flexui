# Architecture

Read this first when resuming work. Requirements are in `/claude.md`, the
supported subset is in `spec/SUBSET.md` (source of truth for behavior), and
current progress is in `spec/STATUS.md`.

## Pipeline

```
HTML ──dom──> node tree ──style──> computed styles ──layout──> boxes
     ──paint──> display list ──raster──> *image.RGBA ──platform──> window/PNG
events <──platform── hit test / focus / handlers (app) ──> state ──> re-render
```

## Packages

Public API (keep small, documented, semver `v0.x`):

| Package | Role |
|---|---|
| `flexui` (repo root) | `RenderHTML`, `CheckHTML`; `App`, `Options`, `Ctx`, `Handler`: `New`, `SetView`, `SetViewFunc`, `Handle`, `Invalidate`, `Bounds`, `Config`, `Warnings`; implements `platform.AppHandler` (Frame renders the view through `internal/view`; Event translates platform events and runs handlers). |
| `platform` | `Platform` / `AppHandler` / `Event` / `Clipboard` interfaces. |
| `platform/darwin` | macOS window via purego: `Run(app)`, `Open`/`Window` (layer presentation, event translation), `Clipboard`; `stages/stageN` verification commands. Compile-checked only. |
| `platform/windows` | Windows window via x/sys/windows: `Run(app)`, `Open`/`Window` (DIB presentation, message translation), `Clipboard`; `stages/stageN`. Compile-checked only. |
| `platform/internal/stagekit` | Shared content of the stage commands (pattern, pages, apps, expectations). |
| `platform/headless` | Reference backend for tests; `Render(app, "out.png")`, `RenderImage`, and `Session` (`Start`, `Click`, `Press`, `Move`, `Wheel`, `Key`, `Resize`, `Frame`) to step synthetic events. |

Implementation (`internal/`, free to change):

| Package | Role | Depends on |
|---|---|---|
| `internal/geom` | `Unit` (fixed-point 1/64 px), `Rect`, `Size`, `Edges`, 128-bit `MulDiv`. | stdlib |
| `internal/style` | Computed `Style` (layout, paint, text props), declaration parser, colors, style sheets, selectors, cascade (`Cascade.Compute`) over a small `Element` interface. | `geom` |
| `internal/layout` | Flexbox engine. Input: tree of `*layout.Node`. Output: `Node.Box`. | `geom`, `style` |
| `internal/paint` | `Canvas` draw interface (incl. `Glyphs` runs over a `GlyphSource`), `List` display list, `Raster` integer-only software rasterizer. | `geom` |
| `internal/dom` | HTML → node tree (`<body>` root, `<style>` collection); `*dom.Node` implements `style.Element`. | `golang.org/x/net/html`, `style` |
| `internal/render` | dom → cascade → layout → paint order (stacking contexts) → display list → raster. `Parse`, `Cache.Style` (sheet and text caches), `Document.Layout/Place/Paint/Draw/HitTest`, `Build`, `Render`. | all of the above |
| `internal/edit` | Single-line text editing model (`Buffer`: graphemes, words, selection). | `go-text/typesetting/segmenter` |
| `internal/view` | Live document between frames: staged invalidation (parse/style/layout/place/paint), retained frame, hit testing, event dispatch (bubbling) to a handler callback; interaction state (M6). | `render`, `dom` |
| `internal/text` | Bundled Inter faces, shaping (font units), line breaking, `Text` layout (lines, intrinsic widths, ellipsis), glyph rasterizer (`paint.GlyphSource`). | `go-text/typesetting`, `geom`, `style`, `paint` |

`examples/` is a separate module (`replace` to `../`) that uses only the
public API, proving the library works for an external consumer:
`hello`, `counter` (`-window` on macOS) and `demo` (M9: list, form, modal;
scripted session in `script.go`, goldens in `demo/testdata`).

Rule: the root package and `internal/...` never import `platform/<os>`
or OS-specific packages; backends depend on the interfaces, never the
reverse. `scripts/check-portability.sh` enforces this and builds both
modules for 5 targets with `CGO_ENABLED=0`.

## Layout engine (`internal/layout/`)

* `layout.go`: public API (`Node`, `Box`, `Layout(root, w, h)`), the
  `compute` dispatcher with a small per-node cache, and leaf sizing.
* `flex.go`: CSS Flexbox §9 algorithm for one container. One function serves
  both sizing (intrinsic measurement) and final layout (`perform`).
* `absolute.go`: absolutely positioned children.
* `compute(node, input)` returns the border-box size. Its inputs:
  * `known`: border-box sizes imposed by the parent.
  * `definite`: whether each known size counts as definite for
    descendants' percentages (§9.8).
  * `ignoreSize`: treat the node's own width/height as auto, to measure its
    content size (automatic minimum size, `flex-basis: content`). Min/max
    are not applied on that axis.
  * `parent`: the containing-block size for percentages (only definite
    sizes are passed).
  * `avail`: the available space per axis (`Definite(v)` / `MinContent` /
    `MaxContent`).
  * `wantW`: the caller needs only the width, so height work is skipped.
  * `perform`: do the final layout of descendants.

  The overall shape resembles Taffy, but it is written from the spec.
* Keeping intrinsic sizing cheap matters: without it, nested auto-sized
  containers are exponential. Three things keep it linear-ish:
  1. **Width-only calls** (`wantW`): a column container's width is the
     widest item (`columnWidth`), a row's width only needs item widths.
  2. **Canonical cache keys** (`Node.canonical`): available height never
     affects results, and width-only calls drop all height inputs.
     Exception: column-wrap containers.
  3. A 12-entry per-node cache of non-perform results, reset per pass.
  `TestDeepNestingCost` guards this.
* Column containers resolve their width before their height. Leaves use
  fit-content width, then measure the height at that width.
* All math is integer `geom.Unit`. Divisions that split space use
  cumulative `MulDiv` so parts sum exactly.

### Testing layout

`internal/layout/cases_test.go` holds table-driven cases written with CSS
declaration strings. Each case is checked two ways:

1. Hand-written `want` rectangles for key elements (where given).
2. Chromium's result, stored in `testdata/layout/chrome.json`. Regenerate it
   with `make crosscheck` (uses node + playwright + headless Chromium, which
   exist in the cloud dev environment). Cases that deliberately deviate from
   CSS set `noChrome` with a reason.

## Conventions

* Determinism: no floats in layout; no map iteration where order matters;
  bundled fonts only; `path`/`io/fs` for paths; `\n` line endings
  (`.gitattributes`).
* Pre-commit: `make check` (gofmt, vet, test, portability).

## Paint (`internal/paint/`)

* `paint.go`: `Color`, `RRect`, `Shadow`, the `Canvas` interface and the
  `List` display list (record, `Replay` into any Canvas). Producers (M3)
  emit logical-unit commands; backends decide snapping and AA, so a GPU
  backend can be added behind the same interface.
* `raster.go`: `Raster` (software Canvas): snapping (`shapeOf`,
  `snapWidth`), coverage-weighted fills, borders as outer-minus-inner
  coverage, clip stack (bounds + rounded clip masks), opacity layers from a
  reused pool. `Begin(dst, Scale64(scale))` / `End()` per frame.
* `shape.go`: snapped device-space rounded rect, radius normalization,
  coverage (exact classification + 8x8 supersampling, 128-bit ellipse
  test).
* `shadow.go`: shadow mask, 3+3 box-blur passes with exact reciprocal
  division, composite outside the box.
* Tests: `golden_test.go` (scenes -> `testdata/paint/*.png`, `-update`),
  `raster_test.go` (snapping invariants, determinism, no allocations).

## Render (`internal/render/`)

* `Build(html, css)`: parse, cascade (css first, then `<style>`s, then
  inline), build `Box` tree with `layout.Node`s; text nodes skipped with a
  warning until M4.
* `Document.Layout(w, h)`: run layout, compute absolute border boxes.
* `Document.Paint(canvas)`: CSS paint order (spec/SUBSET.md §6.4). An
  implicit canvas box is the root stacking context; `context` paints a
  context (own box, negative z, tree-order walk with z=0 contexts inline,
  positive z), `collect` finds out-of-order contexts with the overflow
  clips of their ancestors.
* Text runs become anonymous `Box`es (`Box.Text`, `El == nil`) whose
  layout node measures through `textMeasure` (`layout.Measurer`). They
  paint inside their parent before its element children.
* Tests: `golden_test.go` renders `testdata/render/docs/*.html` (size in
  the first-line comment) to `testdata/render/golden`; `chrome_test.go`
  compares with Chromium screenshots (`testdata/render/chrome`, made by
  `scripts/screenshot.cjs` via `make crosscheck`) and reports max/mean
  error.

## Text (`internal/text/`)

* `font.go`: embedded Inter faces (`ForWeight`), HarfBuzz font at scale =
  unitsPerEm so shaped advances are integers.
* `coverage.go`: workaround for a go-text digest bug (rewrites range
  coverages to glyph lists after parsing; `TestKerning` guards it).
* `layout.go`: `New(raw, Params)` processes white space, shapes, finds
  break segments and pen positions; `MinContent`/`MaxContent`,
  `Layout(width)` (cached last width) → `Block` of `Line`s, `Height`,
  `Glyphs(line)`. The Chromium rounding rules are in `lineSpan` / `span`.
* `mask.go`: glyph outline → coverage mask (integer flattening and
  16-row scanline coverage), cached by (glyph, ppem, quarter-pixel x).
* Chromium comparison: `scripts/flexui-page.cjs` loads the fonts with
  `@font-face` and wraps text runs in `<x-t>` (the anonymous box);
  `scripts/textcheck.cjs` dumps boxes and lines; `internal/render/text_test.go`
  compares (docs in `testdata/text/docs`, 300 seeded random documents in
  `text_random_test.go`).

## App runtime (`flexui.go`, `internal/view/`)

* `App.Frame`: if dirty (a handler ran or `Invalidate`), call the view
  function and `view.SetSource(html, css)`; then `view.Frame`, which
  recomputes only invalidated stages and copies the retained image into
  the platform's buffer.
* `App.Event`: refresh the view first (so events see the current
  document), then translate the platform event into view calls
  (`PointerDown/Up`, ...). The view dispatches DOM-like events along the
  parent chain and calls `App.call` for each element with a matching
  `on<type>` attribute; `call` runs the Go handler and marks the app dirty.
* Paint order and hit order share one traversal (`render.visitor`):
  `painter` draws, `hitter` records the last element containing the point
  inside all active clips.
* Interaction (`internal/view`): `event.go` turns pointer/wheel/key input
  into state changes and events; `state.go` holds element keys (id or
  tree path, for focus and scroll across rebuilds), pseudo-class chains
  (`setHover`/`setActive`/`setFocus` set `dom.Node.Dyn` and invalidate
  style only if `Document.Pseudo` uses the flag), focus order, and
  scrolling (offsets by key, applied in `Document.Place`).
* `View.update` loops at most 3 times: a placement can change what is
  under the pointer, which can change `:hover`, which can restyle.
* Tests: `app_test.go` (counter goldens in `testdata/app`, bubbling,
  caching counters via `view.Stats`), `interact_test.go` (hover, active,
  focus, keyboard, scrolling goldens and behavior),
  `internal/render/hit_test.go` (z-order cases, hit == paint property).

## Inline formatting and replaced elements (M5b)

* `render/inline.go`: `build` groups each maximal run of text and inline
  elements (`isInline`: display inline, inline all the way down) into an
  anonymous box via `inlineBox`, which collects `text.Run`s (text, layout
  style, enclosing inline styles) and creates `Inline` boxes for the
  elements (in `Document.boxes`, not in the flex tree). After placement,
  `placeInline` computes their fragments (per line: horizontal text
  extent, content area of their font). `paintText` draws inline
  backgrounds, then glyph runs per text run in its owner's color. The
  hitter checks fragments.
* `text`: a `Text` is a paragraph of runs; advances are in fine units
  (font units x size) so mixed sizes sum exactly; shaping runs per font,
  widths rounded per run ("pieces", Chromium's inline items); line boxes
  from the strut and the runs (and their enclosing boxes) on each line.
* `render/ua.css`: the default style sheet (embedded; also loaded by the
  Chromium oracle). `style.Rule.UA` puts it below author rules.
* `render/image.go` (`<img>`, `Cache.Assets`, decoded-image cache),
  `paint/image.go` (bilinear `DrawImage`), `layout.AspectRatio`.
* `render/scrollbar.go`: overlay scrollbar geometry (`Box.Thumbs`), drawn
  by the painter inside the container's clip; `view/scrollbar.go` drags
  thumbs.
* `render/input.go`: `<input>` measuring (one line, 160px).

## Text inputs (M8a)

* `internal/edit.Buffer` is the pure model. `view/input.go` keeps one per
  input (`inputs`, by element key), synced with the `value` attribute in
  `syncInputs` after each parse, maps keys/text/pointer to edits, keeps
  the caret visible (`reveal`), and dispatches `input`/`change`.
* Rendering reads the state at paint time through `Document.Input`
  (`render.InputState`), so editing only repaints (no restyle or
  relayout). `render/input.go` draws text, placeholder, selection and
  caret, and offers geometry helpers (`InputCaretX`, `InputContent`).

## Incremental frames

* `paint/damage.go`: `Raster.Damage(prev, next)` diffs two display lists
  into device rectangles (op-by-op, or prefix/suffix splicing when the
  structure changed); `BeginClip` replays a list inside one rectangle.
* `view.paint`: double-buffered lists (`lists[cur]`), damage repaint or
  full repaint, dirty rectangles copied to the platform buffer when it is
  the same image; `verify` / `verifyRestyle` under `FLEXUI_CHECK_DAMAGE`.
* `render/restyle.go`: `Document.Restyle(nodes)` recomputes styles of the
  elements whose pseudo-class state changed, in place, when
  `style.SameLayout` holds for all of them.
