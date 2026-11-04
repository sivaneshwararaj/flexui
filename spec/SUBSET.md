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
* If the base is indefinite (for example `height: 50%` inside an auto-height
  container), the percentage behaves as `auto` (for `min-*`: as `0`; for
  `max-*`: as `none`; for gaps, margins and padding: as `0`).
* For absolutely positioned elements the containing block is the parent's
  **padding box**.
* A size given to a flex item by the flex algorithm counts as definite for
  its children's percentages only as CSS Flexbox §9.8 says. The flexed main
  size is definite if the container's main size is definite. A stretched
  cross size is definite if the container's cross size is definite. A
  container whose size comes from its content is never definite, even
  after layout. So `height: 50%` inside an auto-height column stays
  `auto`.

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
* Distribution fallbacks when free space is negative (overflow), for both
  `justify-content` and `align-content`: `space-between`, `space-around`
  and `space-evenly` all behave as `flex-start`. This is the CSS Box
  Alignment "safe" fallback and matches Chromium. (Flexbox Level 1 said
  `center` for space-around.) `center` and `flex-end` do overflow on the
  start side. With a single item and positive free space, `space-between`
  is `flex-start` and `space-around`/`space-evenly` center it.
* Intrinsic sizes of flex containers (needed when a container is sized by
  its content):
  * max-content main size = sum of the items' hypothetical outer main
    sizes plus gaps (longest line for wrapping containers).
  * min-content main size = sum of the items' min-content contributions
    (largest contribution for wrapping containers).
  * A box whose width is auto and is not stretched gets the fit-content
    width: `max(min-content, min(max-content, available))`, then clamped by
    `min-width`/`max-width`. Its content is then laid out at that width:
    text wraps at the clamped width, but the box keeps the clamped width
    even if the wrapped lines are narrower.
  * A column container's width is resolved before its height (heights
    depend on widths).
  * A column container's auto height is always its content height. It does
    not shrink to the available height, as in CSS block-axis sizing.

### 2.4 Absolute positioning

* Absolutely positioned children do not take part in flex layout.
* Containing block = parent's padding box.
* Width: `width` if set; else `left`+`right` if both set; else
  fit-content within the containing block. Height likewise. If both `top`
  and `bottom` are set and `height` is auto, the height is
  `cb height - top - bottom - margins`.
* Position: `left` if set, else `right`, else the **static position**:
  where the item would sit if it were the only flex item, using the parent's
  `justify-content` (main axis) and the item's `align-self` / parent's
  `align-items` (cross axis). `space-between`/`stretch` → start,
  `space-around`/`space-evenly` → center. Same for `top`/`bottom`.
* `margin: auto` on absolutely positioned elements computes to 0
  (deviation D5).

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
* When space is divided (flex grow/shrink, space-between, ...) it is
  divided with integer arithmetic using cumulative rounding, so the parts
  always sum exactly to the whole. The first items may differ from the last
  ones by 1/64 px.

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
| D5 | `margin: auto` on absolutely positioned elements is 0. | Simplicity. |
| D6 | No baseline alignment. | Not in the subset. |
| D7 | Cascade by source order + pseudo-class layer, not specificity. | Requested simplicity, documented in §3. |
| D8 | `overflow` is one value for both axes. | Simplicity. |
| D9 | Intrinsic (content-based) sizing assumes widths never depend on heights. Exception: a column-wrap container that is itself being measured. When a column-wrap container sits inside a subtree sized by its content width, percentages against ancestor heights are treated as indefinite during that measurement. | Keeps intrinsic sizing linear instead of exponential in tree depth. Only affects column-wrap inside content-sized boxes. |
| D10 | `MeasureFunc` leaves (text, images) are sized as fit-content: width `max(min-content, min(max-content, available))`, then height at that width. | Same as CSS for text; images get their intrinsic size. |

## 5. Verification

Layout behavior is checked against Chromium. `make crosscheck` renders every
layout test case (`internal/layout/cases_test.go`) as HTML in headless Chromium,
using a base style sheet that encodes D1-D4. It writes the measured
rectangles to `testdata/layout/chrome.json`. `go test ./layout` compares the
engine against that file (tolerance 1/64 px) without needing a browser.
Cases that deliberately deviate from Chromium carry hand-written expected
values and a note.
