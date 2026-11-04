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
| `img` | Replaced leaf element; `src` will be resolved through an `io/fs.FS`. **Not loaded yet**: an empty box plus a warning. |
| `body` | The **root element** (see below). |
| `head`, `title`, `meta`, `link`, `style`, `template`, `noscript` | Not rendered. `<style>` contents are style sheets. |
| `script` | Ignored, with a warning (no JavaScript). |
| anything else | Behaves as `div` and emits a warning. |

Documents are parsed with a full HTML5 parser (`golang.org/x/net/html`),
so fragments, missing `<html>`/`<body>` and unclosed tags behave as in
browsers. The `<body>` element is the root of the rendered tree: it fills
the viewport (§2.6) and can be styled with `body { ... }`. `<html>` is not
part of the tree.

Text runs are anonymous leaf nodes. Each run will be a flex item of its
parent (M4). Until M4, non-whitespace text is not rendered and produces
one warning.

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
| `border-width`, `border-{top,right,bottom,left}-width` | `<len>` in px/rem ≥ 0 \| `thin`(1px) \| `medium`(3px) \| `thick`(5px) | `0` | see D3. Computed value is snapped as in browsers: a width ≥ 1px is floored to whole px, a width in (0, 1px) becomes 1px. Layout and paint therefore use the same widths. |
| `box-sizing` | `content-box` \| `border-box` | `content-box` | |
| `top`, `right`, `bottom`, `left` | `<len>` \| `auto` (negative allowed) | `auto` | |
| `overflow` | `visible` \| `hidden` \| `scroll` | `visible` | one value for both axes; `overflow-x/-y` unsupported |
| `z-index` | `<integer>` \| `auto` | `auto` | paint order only; non-`auto` creates a stacking context (§6.4). Hit testing follows it in M6 |

Paint and text properties (`<color>` = `#rgb`, `#rgba`, `#rrggbb`,
`#rrggbbaa`, `rgb()`/`rgba()` in comma or space syntax with numbers or
percentages, the 148 CSS named colors, `transparent`, `currentcolor`;
`hsl()` and other color functions are unsupported):

| Property | Values | Initial | Inherited | Notes |
|---|---|---|---|---|
| `background-color`, `background` | `<color>` | `transparent` | no | `background` accepts a single color only |
| `border-color` | one `<color>` | `currentcolor` | no | one color for all sides (D15) |
| `border-style` | `solid` \| `none` \| `hidden` | (solid) | no | `none`/`hidden` set the widths to 0; other styles warn |
| `border` | `[<width>] [solid\|none] [<color>]` | | no | without a style keyword nothing is drawn, as in CSS |
| `border-radius` | 1-4 `<len>` (TL TR BR BL order) | `0` | no | `%` of the box width (horizontal) and height (vertical); no `/` syntax |
| `border-{top-left,...}-radius` | `<len> [<len>]` | `0` | no | horizontal, vertical |
| `opacity` | `<number>` \| `<percent>` | `1` | no | clamped to 0..1; creates a stacking context when < 1 |
| `box-shadow` | `none` \| `<x> <y> [<blur> [<spread>]] [<color>]` | `none` | no | one outer shadow (D16); color first or last; default `currentcolor` |
| `color` | `<color>` | `black` | yes | |
| `font-family` | any (stored verbatim) | | yes | used from M4 |
| `font-size` | `<len>` in px/rem | `16px` | yes | used from M4 |
| `font-weight` | `normal` \| `bold` \| 1..1000 | `400` | yes | used from M4 |
| `line-height` | `normal` \| `<number>` \| `<len>` | `normal` | yes | used from M4 |
| `text-align` | `left` \| `right` \| `center` | `left` | yes | used from M4 |
| `white-space` | `normal` \| `nowrap` \| `pre` \| `pre-wrap` | `normal` | yes | used from M4 |
| `text-overflow` | `clip` \| `ellipsis` | `clip` | no | used from M4 |

`1rem` is always 16px. The root's font size does not change it, because
`<html>` is not styleable.

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
(`a, b`), pseudo-classes `:hover`, `:focus`, `:active`. Type selectors are
case-insensitive; ids and classes are case-sensitive.

Unsupported selectors (`+`, `~`, attribute selectors, pseudo-elements,
other pseudo-classes such as `:nth-child`) produce a warning, and that
selector is dropped from its list. At-rules (`@media`, `@font-face`, ...)
are skipped with a warning. Invalid declarations in a rule are warned once
per rule, not once per matching element.

Sources, in order: the `css` argument of `RenderHTML` / `App.SetView`,
then the document's `<style>` elements in document order, then each
element's `style=""` attribute.

Cascade order, lowest to highest (no specificity calculation):

1. Initial values.
2. `<style>` rules in **source order** (later rule wins, regardless of
   selector specificity).
3. Rules with a pseudo-class, in source order (so `:hover` overrides the
   base rule written before or after it).
4. Inline `style=""`.

Inheritance: only `color`, `font-family`, `font-size`, `font-weight`,
`line-height`, `text-align` and `white-space` inherit, as in CSS.
`currentcolor` (the initial `border-color` and the default shadow color)
resolves to the element's computed `color`.

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

## 6. Painting and rasterization (M2, M3)

The renderer draws through `paint.Canvas` (fill rounded rect, border,
box shadow, clip, opacity group). Commands are recorded into a display list
(`paint.List`) in logical units. The software rasterizer (`paint.Raster`)
turns them into an `*image.RGBA` using **integer arithmetic only**. The
CSS paint properties (§2.1) map to these commands: `box-shadow`, then
`background-color`, then the border, each on the border box with
`border-radius`. `overflow: hidden | scroll` clips descendants to the
padding box with the inner radii. `opacity < 1` becomes an opacity group.

### 6.1 Pixel-snapping rule

1. Logical lengths (1/64 px) are scaled to device units (1/64 device px).
   The device scale is fixed-point with 1/64 precision: 1, 1.25, 1.5 and 2
   are exact. It is converted from the platform's float once per frame.
2. Every **rectangle edge** is rounded independently to the nearest
   device-pixel boundary, halves rounding up: `x0 = round(x)`,
   `x1 = round(x + w)`. This applies to fills, borders' outer edges, clip
   rects, and shadow shapes after offset and spread. Boxes that touch in
   layout touch exactly in pixels, with no gaps and no double-painted
   seams. A box's pixel size can differ from its layout size by one pixel.
3. **Border widths** round to whole device pixels, with a minimum of one
   pixel for any non-zero width. The inner edge is the snapped outer edge
   minus the snapped width, so a uniform border is equally thick on all
   sides wherever the box lands.
4. **Corner radii, blur and spread are not snapped.** Radii that do not
   fit are scaled down uniformly (CSS Backgrounds §5.5). Inner border radii
   are the outer radii minus the adjacent border widths, so they can be
   elliptical.

### 6.2 Anti-aliasing and compositing

* Straight edges are pixel-aligned (6.1), so only rounded corners need
  anti-aliasing. Corner pixels use 8x8 supersampling (65 coverage levels)
  with exact 128-bit ellipse tests. Pixels entirely inside or outside the
  curve are classified exactly without sampling.
* Blending: see §6.3.
* Clips: rectangle clips intersect bounds. Rounded clips multiply coverage
  (anti-aliased clip edges).
* Opacity groups render into an offscreen layer covering the current clip
  bounds and are composited with the group alpha. Overlaps inside a group
  are therefore not visible, as in CSS `opacity`.
* Box shadow: the shape is the box offset by (x, y) and grown by spread
  (radii grow by spread). Its coverage is blurred by three box-blur passes
  per axis whose combined variance approximates a Gaussian with
  sigma = blur/2 (CSS). The shadow is not painted inside the box itself.

### 6.3 Blending model

Everything below is integer arithmetic and is exactly what
`internal/paint` does, so these formulas define the output bit for bit.

**Color space.** All colors are sRGB, gamma-encoded, 8 bits per
channel. Blending works directly on the encoded values (no conversion to
linear light, no gamma correction), the same as browsers' default
compositing. There is no color management: output pixels are sRGB values
and PNGs are written untagged.

**Input colors.** CSS colors become non-premultiplied `R, G, B, A` in
0..255. Fractional components round to nearest (`rgb(10.5 ...)` → 11,
alpha `0.5` → 128, `50%` → 128), computed with fixed-point decimal
parsing, never floats.

**Target format.** `image.RGBA`: 8-bit **premultiplied** RGBA. The
frame starts as opaque white (the canvas color, §7).

**Rounding helper.** `div255(x) = (x + 128 + ((x + 128) >> 8)) >> 8`,
which equals `round(x / 255)` (halves up) for all `0 <= x <= 255*255`
(tested exhaustively; it is not exact far beyond that, and no call site
exceeds it). Every division by 255 goes through it.

**Coverage.** A pixel's geometric coverage `k` is in 0..64: 64 inside, 0
outside, the count of 8x8 samples at rounded corners. Rounded clips
multiply it: `k = (k * clipK) >> 6` for each active rounded clip
(truncating). The effective source alpha is `a = (A*k + 32) >> 6`, i.e.
`round(A*k/64)`.

**Source-over** (each fill, border and shadow pixel), with `inv = 255 - a`:

```
dst.C = div255(src.C * a + dst.C * inv)     for C in R, G, B
dst.A = a + div255(dst.A * inv)
```

The premultiplication of the source and the attenuation of the
destination share one rounding step. With `a = 255` this writes the
source color exactly.

**Shadows.** The blurred mask value `v` is in 0..65280 (64 coverage
steps × 1020). The effective alpha is
`a = (A * v * k + 65280*32) / (65280*64)` (rounded), where `k = 64 - box
coverage` (the shadow is not drawn under its box). The box blur divides
with exact reciprocal multiplication, rounding halves up.

**Opacity groups.** A group is drawn into a transparent premultiplied
layer, then composited with group alpha `g` (0..255; CSS
`opacity` → `round(opacity*255)`):

```
sa    = div255(layer.A * g)
dst.C = div255(layer.C * g) + div255(dst.C * (255 - sa))
dst.A = sa + div255(dst.A * (255 - sa))
```

The two terms are rounded separately, so a group can differ by 1 from
painting the same shape directly at the combined alpha. Overlaps inside a
group composite before the group alpha, as in CSS.

| # | Deviation | Reason |
|---|---|---|
| D15 | One `border-color` for all four sides. | Not needed for app UIs yet; easy to extend. |
| D16 | Only one, outer `box-shadow` (no `inset`, no lists). | Subset. |
| D17 | Shadow blur is a 3-pass box blur (a close approximation of the Gaussian); pixels may differ slightly from browsers. | Integer-only, fast, deterministic. |
| D18 | Curved edges (rounded corners, rounded clips) are anti-aliased with 8x8 supersampling. Browsers (Skia) use analytic coverage, so individual edge pixels can differ by up to ~85 of 255 levels. Straight edges are identical. | Exact integer coverage, deterministic. |
| D19 | Group opacity composites the layer with separately rounded terms (§6.3); pixels can differ from browsers by 1-2 levels. | Integer-only. |
| D20 | The canvas is always white: `body`'s background is not propagated to the canvas. Because `body` fills the viewport, this is visible only through `body` margins. | Simplicity; app windows. |

### 6.4 Paint order and stacking contexts

Every element is positioned (D2), which simplifies CSS 2 Appendix E:

* A **stacking context** is formed by the canvas (the root context) and by
  every element with `z-index` other than `auto` or with `opacity < 1`.
  `<body>` is an ordinary element inside the canvas context, as in
  browsers, where the root context belongs to `<html>`.
* Inside a stacking context, in order:
  1. The context element's own shadow, background and border.
  2. Descendant contexts with **negative** `z-index`, lowest first (stable
     tree order for ties).
  3. All other descendants in **tree order**: each element's shadow,
     background and border, followed by its descendants. Descendant
     contexts with `z-index: 0`, or `auto` with `opacity < 1`, are painted
     here at their tree position, atomically.
  4. Descendant contexts with **positive** `z-index`, lowest first.
* A context's descendants never interleave with things outside it. For
  example, a `z-index: 999` child of a `z-index: 1` box stays below a
  `z-index: 2` sibling of that box.
* `overflow` clips apply to all descendants, including stacking contexts
  painted out of tree order. Each such context is painted with the clips of
  its ancestors between it and its parent context.
* A negative `z-index` box whose parent context is the canvas paints
  behind `body`'s background, and is hidden if `body` has one, as in
  browsers.

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
* Painting is checked with golden PNGs (`testdata/paint`,
  `go test ./internal/paint -run TestGolden -update` to regenerate; look at
  the images before committing). Unit tests cover the snapping invariants
  and check that steady-state frames allocate nothing.
* RenderHTML output is checked against **Chromium screenshots**
  (`internal/render/chrome_test.go`). The documents in
  `testdata/render/docs` (non-text, 13 documents, 3 also at 2x) are
  screenshotted by `scripts/screenshot.cjs` (`make crosscheck`) with a
  base sheet encoding D1-D4 and D20. For each one the test reports the max
  error, the mean absolute channel error, and the share of pixels off by
  more than 8 and 32 levels. The default bound is near-exact: mean ≤ 0.05
  and no pixel off by more than 32. Documents above that are listed with
  their deviation (D17 shadows, D18 curve AA, D19 opacity rounding) and
  bounds about 1.5x their measured error. Current numbers are in
  `spec/STATUS.md`.
* `go test ./internal/layout -run TestReduce -v -args -reduce=CASE`
  minimizes a failing case against a live Chromium
  (`scripts/chrome-serve.cjs`). It deletes subtrees and declarations while
  the mismatch persists, which turns a random tree into a 2-4 node
  reproduction.
