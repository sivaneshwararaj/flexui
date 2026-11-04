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

| Package | Role | Depends on |
|---|---|---|
| `geom` | `Unit` (fixed-point 1/64 px), `Rect`, `Size`, `Edges`, 128-bit `MulDiv`. | stdlib |
| `style` | Computed `Style`, value types, declaration parser for the subset, warnings. Selectors/cascade in M3. | `geom` |
| `layout` | Flexbox engine. Input: tree of `*layout.Node` (style + children + optional measure func). Output: `Node.Box`. | `geom`, `style` |
| `dom` | HTML → node tree (M3). | `golang.org/x/net/html` |
| `paint` | Display list + software rasterizer (M2). | `geom` |
| `text` | Font, shaping, line breaking (M4). | `go-text/typesetting` |
| `app` | Handlers, state, re-render loop, events (M5). | all core |
| `platform` | `Platform` / `AppHandler` interfaces. | stdlib |
| `platform/headless` | Reference platform for tests. | `platform` |

Rule: core packages (`geom style layout dom paint text app`) never import
`platform/*` or OS-specific packages. `scripts/check-portability.sh`
enforces this.

## Layout engine (`layout/`)

* `layout.go`: public API (`Node`, `Box`, `Layout(root, w, h)`), the
  `compute` dispatcher with a small per-node cache, and leaf sizing.
* `flex.go`: CSS Flexbox §9 algorithm for one container. One function serves
  both sizing (intrinsic measurement) and final layout (`perform`).
* `absolute.go`: absolutely positioned children.
* `compute(node, input)` takes the node's *known* border-box sizes (imposed
  by the parent), the containing-block size for percentages, the available
  space per axis (`Definite(v)` / `MinContent` / `MaxContent`), and whether
  to perform final layout. It returns the border-box size. This mirrors
  Taffy's design but is written from the spec.
* All math is integer `geom.Unit`. Divisions that split space use
  cumulative `MulDiv` so parts sum exactly.

### Testing layout

`layout/cases_test.go` holds table-driven cases written with CSS
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
