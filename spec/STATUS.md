# Status

| Milestone | State |
|---|---|
| M0 Spec and scaffolding | done |
| M1 Layout engine | done |
| M2 Rasterizer | done |
| M3 HTML+CSS to pixels | not started |
| M4 Text | not started |
| M5 Handlers and state | not started |
| M6 Interaction | not started |
| M7 Portability gate + macOS | portability script exists and passes; darwin not started |
| M8 Text input | not started |
| M9 Demo app | not started |

## Works

* `style`: declaration parsing for all layout properties of the subset,
  shorthands (`margin`, `padding`, `border-width`, `gap`, `flex`), px/%/rem,
  warnings for unsupported input. No floats: decimals are parsed straight
  into fixed point.
* `layout`: the full flexbox algorithm in `spec/SUBSET.md` §2:
  row/column (+reverse), wrap/wrap-reverse, grow/shrink/basis with the
  freeze loop, automatic minimum sizes, min/max, justify-content,
  align-items/self/content, auto margins, gap, padding/border/box-sizing,
  percentages with §9.8 definiteness, intrinsic (fit-content /
  min-content / max-content) sizing, relative and absolute positioning,
  `display: none`, scrollable-overflow extents (`Box.ContentW/H`).
* Layout is verified against Chromium: 111 hand-written cases plus 2400
  seeded random trees (`internal/layout/random_test.go`), reference data in
  `testdata/layout/chrome.json` (`make crosscheck`). All match to 1/64 px
  except 25 random cases (about 1%), which are listed with deviation
  numbers D11-D14 in `spec/SUBSET.md`. A reducer
  (`TestReduce -reduce=CASE`) shrinks any mismatch to a minimal tree
  against a live Chromium.
* Cost: intrinsic sizing is linear-ish in tree size (about 7 measure calls
  per leaf on a 10-deep tree of auto-sized nested rows/columns).
* `paint`: `Canvas` interface, display list (`List`), integer-only software
  rasterizer (`Raster`): rects, rounded rects (elliptical, normalized),
  borders (per-side widths, elliptical inner radii), single outer box
  shadow with blur/spread/offset, rect and rounded clips, opacity groups,
  HiDPI scale. Pixel snapping as in `spec/SUBSET.md` §6. 8 golden PNGs,
  snapping/tiling invariant tests, zero allocations per steady-state frame.
  A 320x200 card frame at 2x (shadow, clip, border) renders in ~2 ms.
* Determinism: integer-only layout; tests also pass with `GOARCH=386`.
* `platform`: interface; `headless` platform delivers events and renders
  frames into an image.
* `scripts/check-portability.sh`: builds and vets 5 targets with
  `CGO_ENABLED=0` and checks the core import rule.

## Stubbed

* `internal/dom`, `internal/text`: package docs only.
* No style cascade, selectors or paint properties in `style` yet (M3);
  layout tests build styles from declaration strings, paint tests call the
  Canvas directly.
* The public `flexui.App.Frame` only clears to white until M3.

## Known limitations / bugs

* Layout deviations D11-D14 (`spec/SUBSET.md` §4): absolute boxes with
  both vertical insets that overflow, intrinsic sizes of wrapping
  containers with explicit `flex-basis`, some nested cyclic percentages,
  and 2/64 px rounding differences.
* `layout` allocates one item slice per container per pass. Fine for now;
  reuse buffers if profiling shows it matters.
* The root `flexui.App` does not render views yet (M3) or run handlers
  (M5).
* Rasterizer speed: the shadow blur dominates (~half of a card frame).
  Possible later wins: skip blurring the region hidden under the box,
  span-based coverage for borders and clips.
