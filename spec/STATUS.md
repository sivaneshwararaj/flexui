# Status

| Milestone | State |
|---|---|
| M0 Spec and scaffolding | done |
| M1 Layout engine | done |
| M2 Rasterizer | not started |
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
* Layout is verified against Chromium: 106 cases in
  `layout/cases_test.go`, reference data in `testdata/layout/chrome.json`
  (`make crosscheck`). All match to 1/64 px. Hand-written expectations
  cover the basic cases too.
* Cost: intrinsic sizing is linear-ish in tree size (about 7 measure calls
  per leaf on a 10-deep tree of auto-sized nested rows/columns). All 106
  cases lay out in ~0.5 ms total.
* Determinism: integer-only layout; tests also pass with `GOARCH=386`.
* `platform`: interface; `headless` platform delivers events and renders
  frames into an image.
* `scripts/check-portability.sh`: builds and vets 5 targets with
  `CGO_ENABLED=0` and checks the core import rule.

## Stubbed

* `dom`, `paint`, `text`, `app`: package docs only.
* No style cascade or selectors yet (M3); tests build styles from
  declaration strings.

## Known limitations / bugs

* Deviation D9 (spec): width-only intrinsic sizing does not see heights,
  which matters only for column-wrap containers inside content-sized boxes.
  Not covered by tests.
* `layout` allocates one item slice per container per pass (~600 allocs for
  the whole test suite). Fine for now; reuse buffers if profiling shows it
  matters.
* Percentage gaps against an indefinite container size resolve to 0 (CSS
  resolves them against the final size for layout).
