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
| `flexui` (repo root) | `App`, `Options`, `Ctx`, `Handler`: `New`, `SetView`, `SetViewFunc`, `Handle`, `Config`; implements `platform.AppHandler`. |
| `platform` | `Platform` / `AppHandler` / `Event` / `Clipboard` interfaces. |
| `platform/headless` | Reference backend for tests; `Render(app, "out.png")`, `RenderImage`. |

Implementation (`internal/`, free to change):

| Package | Role | Depends on |
|---|---|---|
| `internal/geom` | `Unit` (fixed-point 1/64 px), `Rect`, `Size`, `Edges`, 128-bit `MulDiv`. | stdlib |
| `internal/style` | Computed `Style`, value types, declaration parser, warnings. Selectors/cascade in M3. | `geom` |
| `internal/layout` | Flexbox engine. Input: tree of `*layout.Node`. Output: `Node.Box`. | `geom`, `style` |
| `internal/paint` | `Canvas` draw interface, `List` display list, `Raster` integer-only software rasterizer. | `geom` |
| `internal/dom` | HTML → node tree (M3). | `golang.org/x/net/html` |
| `internal/text` | Font, shaping, line breaking (M4). | `go-text/typesetting` |

`examples/` is a separate module (`replace` to `../`) that uses only the
public API, proving the library works for an external consumer.

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
