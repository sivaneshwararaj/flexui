# flexui HTML/CSS subset

This document is the source of truth for what flexui supports. Anything not
listed here is unsupported. Unsupported properties, values, selectors and
elements produce a warning (`style.Warning`) and are ignored. They never
silently change behavior.

flexui is "web-like, not the web". Where we deviate from CSS on purpose, the
deviation is listed under **Deviations** with the reason.

## 1. Elements

| Element | Behavior |
|---|---|
| `div`, `span`, `p`, `h1`, `h2`, `h3`, `ul`, `li` | Flex container. No built-in margins, bullets or font changes unless a later milestone adds a default style sheet (to be listed here). |
| `button` | Flex container; clickable; focusable (M5/M6). |
| `a` | Flex container; clickable; focusable. No navigation, `href` is ignored. |
| `input` (text only) | Single-line text field (M8). Other `type`s warn and are treated as `text`. |
| `img` | Replaced leaf element; `src` is resolved through an `io/fs.FS` (M3+). |
| anything else | Behaves as `div` and emits a warning. |

Text runs are anonymous leaf nodes. Each run is a flex item of its parent
(see M4).

## 2. Layout model

* Every element is a flex container (`display: flex`). There is no normal
  flow, inline formatting, floats, inline-block, tables, grid, or margin
  collapsing.
* `display` accepts `flex` (initial) and `none`. Any other value (`block`,
  `inline`, `grid`, ...) warns and is ignored.
* Initial `flex-direction` is **`column`** (CSS: `row`).
* Initial `position` is **`relative`** (CSS: `static`). So every element is a
  containing block for its absolutely positioned children (same as React
  Native / Yoga).

### 2.1 Properties

Value syntax: `<len>` is `<number>px`, `<number>%`, `<number>rem`, or `0`.
Numbers are plain decimals (`12`, `-3.5`, `.25`). Exponents, `calc()`, `em`,
`vw`, etc. are unsupported.

| Property | Values | Initial | Notes |
|---|---|---|---|
| `display` | `flex` \| `none` | `flex` | |
| `position` | `relative` \| `absolute` | `relative` | `static`, `fixed`, `sticky` unsupported |
| `flex-direction` | `row` \| `row-reverse` \| `column` \| `column-reverse` | `column` | |
| `flex-wrap` | `nowrap` \| `wrap` \| `wrap-reverse` | `nowrap` | |
| `flex-grow` | `<number>` ≥ 0 | `0` | precision 0.001 |
| `flex-shrink` | `<number>` ≥ 0 | `1` | precision 0.001 |
| `flex-basis` | `<len>` ≥ 0 \| `auto` | `auto` | `content` unsupported |
| `flex` | `none` \| `auto` \| `<grow> [<shrink>] [<basis>]` \| `<basis>` | | omitted basis = `0` (CSS rule) |
| `justify-content` | `flex-start` \| `flex-end` \| `center` \| `space-between` \| `space-around` \| `space-evenly` | `flex-start` | `start`, `end`, `left`, `right`, `normal`, `stretch` unsupported |
| `align-items` | `stretch` \| `flex-start` \| `flex-end` \| `center` | `stretch` | `baseline` unsupported |
| `align-self` | `auto` \| align-items values | `auto` | |
| `align-content` | `stretch` \| `flex-start` \| `flex-end` \| `center` \| `space-between` \| `space-around` \| `space-evenly` | `stretch` | CSS `normal` behaves as `stretch`; we name it `stretch` |
| `gap` | `<len> [<len>]` ≥ 0 | `0` | row-gap then column-gap |
| `row-gap`, `column-gap` | `<len>` ≥ 0 | `0` | |
| `width`, `height` | `<len>` ≥ 0 \| `auto` | `auto` | `min-content`, `fit-content` etc. unsupported |
| `min-width`, `min-height` | `<len>` ≥ 0 \| `auto` | `auto` | `auto` = automatic minimum size (§2.3) |
| `max-width`, `max-height` | `<len>` ≥ 0 \| `none` | `none` | |
| `margin`, `margin-{top,right,bottom,left}` | `<len>` \| `auto` (negative allowed) | `0` | 1-4 value shorthand |
| `padding`, `padding-{top,right,bottom,left}` | `<len>` ≥ 0 | `0` | 1-4 value shorthand |
| `border-width`, `border-{top,right,bottom,left}-width` | `<len>` in px/rem ≥ 0 \| `thin`(1px) \| `medium`(3px) \| `thick`(5px) | `0` | see deviation D3 |
| `box-sizing` | `content-box` \| `border-box` | `content-box` | |
| `top`, `right`, `bottom`, `left` | `<len>` \| `auto` (negative allowed) | `auto` | |
| `overflow` | `visible` \| `hidden` \| `scroll` | `visible` | one value for both axes; `overflow-x/-y` unsupported |
| `z-index` | `<integer>` \| `auto` | `auto` | paint order only (M6) |

Paint properties (`background-color`, `border-color`, `border-radius`,
`opacity`, `box-shadow`, `color`, `font-*`, `line-height`, `text-align`,
`text-overflow`, `white-space`) are specified when they are implemented
(M2-M4) and are currently warned as unsupported.

CSS-wide keywords (`inherit`, `initial`, `unset`, `revert`) and
`!important` are unsupported (warn; `!important` is stripped and the
declaration still applies).

### 2.2 Percentages

* `width`, `min-width`, `max-width`, `left`, `right`, `column-gap`
  resolve against the containing block's **content-box width**; the
  vertical equivalents against its content-box height.
* `margin` and `padding` (all four sides) resolve against the containing
  block's content-box **width**, as in CSS.
* `flex-basis` resolves against the container's inner main size.
* **Horizontal percentages** (`width`, `min/max-width`, `left`/`right`,
  `column-gap`, row-direction `flex-basis`, and all margins and padding)
  resolve against the containing block's width **whenever that width is
  known**, even if it was derived from content ("cyclic percentages", as in
  CSS). While that width is still being computed (intrinsic sizing) they
  count as `auto` (sizes), `0` (margins, padding, gaps) or `none` (max-*).
* **Vertical percentages** (`height`, `min/max-height`, `top`/`bottom`,
  `row-gap`, column-direction `flex-basis`) resolve only against a
  **definite** height, otherwise they behave the same way as above. A
  height is definite if:
  * it is set in style and resolvable, or it is the viewport or an
    absolutely positioned box with both `top` and `bottom`;
  * it is a flexed main size and the flex container's main size is
    definite, or the item has a definite `flex-basis` of its own, or
    `flex-basis: auto` with a definite `height` (CSS Flexbox §9.8 rule 2,
    and Chromium's extension for definite bases);
  * it is a stretched cross size (§9.8 rule 3: once line cross sizes are
    known, stretched items are definite, even in auto-sized containers).
  A height that came from content is not definite. So `height: 50%` inside
  an auto-height column behaves as `auto`, but a percentage `row-gap` in a
  column whose height is definite under the rules above resolves against
  its final height.
* For absolutely positioned elements the containing block is the parent's
  **padding box**, which is always definite.
* `align-self: stretch` applies only when the cross-size property is
  literally `auto`. An unresolvable percentage is not `auto`, so the item
  keeps its content size.

### 2.3 Flex algorithm

We follow CSS Flexible Box Layout Level 1 §9, including:

* Flex base size and hypothetical main size (§9.2), with `flex-basis: auto`
  falling back to `width`/`height` and then to the max-content size. A
  percentage `flex-basis` against an indefinite container size behaves as
  `content` (the content size, ignoring `width`/`height`). The flex base
  size ignores min/max; they apply to the hypothetical main size.
* Automatic minimum size (`min-width/height: auto`) of flex items in the main
  axis = min(specified size, min-content size), clamped by the max size. It is
  `0` for items with `overflow` other than `visible`. In the cross axis it is
  `0`.
* Line breaking for `wrap` / `wrap-reverse`.
* Resolving flexible lengths with the freeze loop, including the
  sum-of-factors-below-1 rule and scaled shrink factors.
* Cross sizes, `stretch`, `align-content`, auto margins on both axes.
* Distribution of free space by `justify-content` / `align-content`
  matches Chromium's integer steps: each gap gets `free / n` truncated to
  1/64 px (n = items-1, items or items+1 for between/around/evenly), and
  the remainder stays at the end. In reversed containers the positions
  are computed from the physical start (the last item first), so the
  remainder ends up at the physical end.
* Overflow (negative free space) fallbacks: `space-between` → `flex-start`;
  `space-around` / `space-evenly` → the writing-mode start (top/left),
  which is `flex-end` in a reversed container (CSS Box Alignment's "safe
  center"). `center` and `flex-end` overflow on the start side. A single
  item: `space-between` → `flex-start`, `space-around`/`space-evenly`
  center it.
* Auto margins share positive free space exactly (cumulative rounding).
* Intrinsic sizes of flex containers (when a container is sized by its
  content). These follow **Chromium**, which differs from the "desired
  flex fraction" algorithm in CSS Flexbox §9.9.1:
  * An item's max-content (min-content) contribution is its definite
    preferred main size if it has one, else its max-content (min-content)
    size, clamped by min/max.
  * In a single-line container, an item with an explicit `flex-basis`
    length that cannot grow (`flex-grow: 0`) contributes at most its
    hypothetical main size. If it cannot shrink (`flex-shrink: 0`), it
    contributes at least that size.
  * Row container: max-content width = sum of max-content contributions
    plus gaps, for wrapping containers too. Min-content width = sum of
    min-content contributions (nowrap) or the largest one (wrap).
  * A box whose width is auto and is not stretched gets the fit-content
    width `max(min-content, min(max-content, available))`, then clamped by
    `min-width`/`max-width`. Its content is laid out at that width: text
    wraps at the clamped width, but the box keeps the clamped width even if
    the wrapped lines are narrower.
  * Width first: every container whose width is unknown determines it with
    a width-only query, then lays out its content at that width.
  * A column container's auto height is its content height. It never
    shrinks to the available height. Column-wrap containers break lines
    only at a definite height, or at their specified height or
    `max-height`, even while their content size is being measured. A
    height that merely came from content does not break lines.
  * A column-wrap container's min-content width is its widest item (lines
    are not broken for that measurement). Its max-content width sums its
    columns. Non-stretched items in a column-wrap container are re-fitted
    to fit-content within their line's cross size.

### 2.4 Absolute positioning

* Absolutely positioned children do not take part in flex layout.
* Containing block = parent's padding box.
* Width: `width` if set, else `left`+`right` if both are set (stretch),
  else fit-content. The available width for fit-content is the containing
  block minus the insets and margins, and minus the padding on the side
  that the static position is anchored to (CSS 2 §10.3.7).
* Height: likewise, except that when both `top` and `bottom` are set and
  `height` is auto, `align-self: flex-start | center | flex-end` makes the
  height fit-content. The box is then aligned inside the inset-modified
  containing block (CSS Position 3). `auto`/`stretch` stretch it. A
  negative-size inset-modified containing block is treated as zero size at
  `top` (see D11 for overflow cases).
* Auto margins (with both insets set and a definite size) absorb the
  remaining space and center the box, as in CSS. If horizontal space is
  negative, the right margin gives way.
* Position: `left` if set, else `right`, else the **static position**:
  where the item would sit if it were the only flex item, using the
  parent's `justify-content` (main axis) and the item's `align-self` /
  parent's `align-items` (cross axis). `space-between`/`stretch` → start,
  `space-around`/`space-evenly` → center. Same for `top`/`bottom`.

### 2.5 Relative positioning

`top`/`left` (or, if those are `auto`, `bottom`/`right`) shift the element
visually after layout without affecting siblings.

### 2.6 Root element and viewport

The root element's containing block is the viewport. If the root's `width`
or `height` is `auto`, it fills the viewport minus its margins in that axis
(like an app window, not like CSS where the root's height is
content-sized).

### 2.7 Numbers and determinism

* All lengths are fixed-point integers in 1/64 px (`geom.Unit`), the same
  resolution as browsers' layout units. Parsing converts decimals to fixed
  point without using floats. Percentages keep 0.001% precision, flex
  factors 0.001.
* Layout does not snap to whole pixels. Positions can be fractional
  multiples of 1/64 px. Paint decides how to rasterize them.
* Flexible lengths (grow/shrink) and auto margins are divided with
  integer arithmetic using cumulative rounding, so the parts always sum
  exactly to the whole. `justify-content`/`align-content` spacing uses
  truncated per-gap steps like browsers (§2.3).

## 3. Selectors and cascade (M3)

Supported: type (`div`), class (`.a`), id (`#a`), universal (`*`),
compound (`div.a#b`), descendant (`a b`), child (`a > b`), selector lists
(`a, b`), pseudo-classes `:hover`, `:focus`, `:active`.

Cascade order, lowest to highest (no specificity calculation):

1. Initial values.
2. `<style>` rules in **source order** (later rule wins, regardless of
   selector specificity).
3. Rules with a pseudo-class, in source order (so `:hover` overrides the
   base rule written before or after it).
4. Inline `style=""`.

Inheritance: only text properties (`color`, `font-*`, `line-height`,
`text-align`, `white-space`) inherit, as in CSS. Details are added in M3/M4.

## 4. Deviations from CSS (deliberate)

| # | Deviation | Reason |
|---|---|---|
| D1 | Initial `flex-direction` is `column`. | Matches block-like stacking for app UIs. |
| D2 | Initial `position` is `relative`; every element is a containing block for absolute children. | Simpler mental model; no `static`. |
| D3 | `border-width` takes effect without a `border-style` (all borders are solid). | No `border-style` in the subset. |
| D4 | Root `auto` width/height fill the viewport. | App windows, not documents. |
| D5 | *Removed:* auto margins on absolutely positioned elements now behave as in CSS (§2.4). | |
| D6 | No baseline alignment. | Not in the subset. |
| D7 | Cascade by source order + pseudo-class layer, not specificity. | Requested simplicity, documented in §3. |
| D8 | `overflow` is one value for both axes. | Simplicity. |
| D9 | *Removed:* width-only measurements no longer drop height information in subtrees that contain a column-wrap container (the only case where widths depend on heights). | |
| D10 | `MeasureFunc` leaves (text, images) are sized as fit-content: width `max(min-content, min(max-content, available))`, then height at that width. | Same as CSS for text; images get their intrinsic size. |
| D11 | An absolutely positioned box with both `top` and `bottom` set may be placed differently from Chromium when it does not fit its inset-modified containing block. This happens when that block has negative size, or when the box overflows it while `align-self` is not stretch or `height` is set. We place it at `top` plus the alignment offset, with no overflow correction. | Chromium's overflow handling here did not fit any consistent rule we could derive from 12 reduced cases. Rare in practice. |
| D12 | Intrinsic (content-based) size of a **wrapping** container whose items have an explicit `flex-basis` can differ from Chromium. | Chromium's contribution rule for multi-line containers with explicit bases (7 reduced cases) does not match the single-line rule of §2.3. Not yet characterised. |
| D13 | Cyclic percentages: percentage heights / bases inside a container whose height is content-derived but definite by §9.8 can resolve against a different intermediate size than Chromium in nested wrap cases. | 3 reduced cases; Chromium relayouts in a different order. |
| D14 | Sub-pixel rounding: up to 2/64 px differences in some `space-*` / percentage combinations. | Chromium mixes float and fixed-point steps. |

### 4.1 Previously documented limitations, now resolved

* **Percentage gaps** against content-sized containers used to resolve to 0
  in all cases. They now follow the percentage rules in §2.2 and match
  Chromium (`gap_percent_*` cases): 0 while the size is being computed, then
  resolved against the final width, or the final height if it is definite.
* **D9** (width-only measurement ignoring heights) is removed; see the
  table above.

## 5. Verification

Layout behavior is checked against Chromium:

* `internal/layout/cases_test.go`: hand-written cases (some with
  hand-computed expectations).
* `internal/layout/random_test.go`: 2400 seeded random trees drawn from the
  whole property pool (randomized differential testing). Mismatches that
  are deliberately not fixed are listed in `knownRandomMismatches` with
  their deviation number (D11-D14). The test fails if a listed case starts
  matching, so the list cannot go stale.
* `make crosscheck` renders all cases as HTML in headless Chromium, using a
  base style sheet that encodes D1-D4, and writes the rectangles to
  `testdata/layout/chrome.json`. `go test` compares against that file
  (tolerance 1/64 px) without needing a browser.
* `go test ./internal/layout -run TestReduce -v -args -reduce=CASE`
  minimizes a failing case against a live Chromium
  (`scripts/chrome-serve.cjs`). It deletes subtrees and declarations while
  the mismatch persists, which turns a random tree into a 2-4 node
  reproduction.
