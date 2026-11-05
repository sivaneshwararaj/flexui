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
| `div`, `p`, `h1`, `h2`, `h3`, `ul`, `li` | Flex container. No built-in margins, bullets or font changes. |
| `span`, `strong`, `b`, `em`, `i` | **Inline** (`display: inline` in the default style sheet): part of an inline formatting context with the surrounding text (§7.1). `strong`/`b` are bold; `em`/`i` are upright (D30: no italic face). |
| `a` | Inline; colored by the default style sheet; clickable and focusable. No navigation, `href` is ignored. |
| `br` | Forced line break inside inline content. |
| `button` | Flex container (row, centered) with a default style (§1.1); clickable; focusable; `disabled` supported. |
| `input` (text only) | Single-line text field with a default style; one line of text tall, 160px wide (§1.3); editing from M8. `disabled` supported. Other `type`s warn and are treated as `text`. |
| `img` | Replaced leaf element showing a PNG or JPEG (§1.2). |
| `body` | The **root element** (see below). |
| `head`, `title`, `meta`, `link`, `style`, `template`, `noscript` | Not rendered. `<style>` contents are style sheets. |
| `script` | Ignored, with a warning (no JavaScript). |
| anything else | Behaves as `div` and emits a warning. |

Documents are parsed with a full HTML5 parser (`golang.org/x/net/html`),
so fragments, missing `<html>`/`<body>` and unclosed tags behave as in
browsers. The `<body>` element is the root of the rendered tree: it fills
the viewport (§2.6) and can be styled with `body { ... }`. `<html>` is not
part of the tree.

Text and inline elements form anonymous inline boxes, each a flex item of
its parent (§7.1).

Documents are always laid out in **standards mode** (no quirks), whether
or not they have a doctype.

### 1.1 Default style sheet

flexui has a small user-agent style sheet, `internal/render/ua.css`
(listed there in full). Its rules lose to every author rule, whatever
their specificity (CSS cascade origins), and to inline styles. It makes
`span, a, strong, b, em, i, br` inline, `strong`/`b` bold, `a` blue, and
gives `button` and `input` a light bordered look with `:hover`, `:active`,
`:focus` and `:disabled` states. **D27**: these defaults are flexui's own,
not browsers' (which differ by platform); the Chromium oracle loads the
same sheet.

### 1.2 Images

* `<img src>` is a `data:` URL (base64 or percent-encoded) or a path in
  `Options.Assets` (an `io/fs.FS`; a leading `/` is ignored). Other URLs
  and missing files warn and leave an empty box. PNG and JPEG are decoded
  with Go's standard library (deterministic, integer).
* Natural size: one image pixel per logical pixel. The image is a
  measured leaf with its natural width as min- and max-content, its
  height following the width through the aspect ratio, and its width
  following a definite height (`layout.AspectRatio`). Images fill their
  content box (no `object-fit`).
* Drawing: §6.5. Content is clipped to the rounded padding box when
  `border-radius` is set. **D28**: Chromium (with the oracle's base
  sheet) does not clip the image to the radius; CSS Backgrounds §5.3 says
  it should, so flexui does. **D29**: JPEG decoding (Go vs.
  libjpeg-turbo chroma upsampling) and the scaling filter differ from
  Chromium by a few levels; layout is identical.

### 1.3 Inputs

An `<input>` is a measured leaf: one line of its font tall, with 160px as
its min- and max-content width, so a flex row does not shrink it below
that (as Chromium does).


## 2. Layout model

* Every element is a flex container (`display: flex`), except inline
  elements inside text (§7.1). There is no normal
  flow, floats, inline-block, tables, grid, or margin
  collapsing.
* `display` accepts `flex` (initial), `inline` (§7.1) and `none`. Any
  other value (`block`, `grid`, ...) warns and is ignored.
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
| `display` | `flex` \| `inline` \| `none` | `flex` | | `inline`: §7.1 |
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
| `font-family` | any (stored verbatim) | | yes | every family renders with the bundled Inter (§7.3) |
| `font-size` | `<len>` in px/rem | `16px` | yes | §7 |
| `font-weight` | `normal` \| `bold` \| 1..1000 | `400` | yes | §7 |
| `line-height` | `normal` \| `<number>` \| `<len>` | `normal` | yes | §7 |
| `text-align` | `left` \| `right` \| `center` \| `start` \| `end` | `start` | yes | §7; `start`/`end` follow `direction` |
| `direction` | `ltr` \| `rtl` | `ltr` | yes | §7.6; also set by the `dir` attribute; flips the flex axes (§7.6) |
| `white-space` | `normal` \| `nowrap` \| `pre` \| `pre-wrap` | `normal` | yes | §7 |
| `text-overflow` | `clip` \| `ellipsis` | `clip` | no | §7 |

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
(`a, b`), pseudo-classes `:hover`, `:focus`, `:active`, `:disabled`. Type selectors are
case-insensitive; ids and classes are case-sensitive.

Unsupported selectors (`+`, `~`, attribute selectors, pseudo-elements,
other pseudo-classes such as `:nth-child`) produce a warning, and that
selector is dropped from its list. At-rules (`@media`, `@font-face`, ...)
are skipped with a warning. Invalid declarations in a rule are warned once
per rule, not once per matching element.

Sources, in order: the `css` argument of `RenderHTML` / `App.SetView`,
then the document's `<style>` elements in document order, then each
element's `style=""` attribute.

Cascade order, lowest to highest:

1. Inherited or initial values.
1. The default (user-agent) style sheet (§1.1), by specificity then
   source order.
2. Matching rules by ascending **specificity**, ties broken by source
   order (later wins). Specificity is the CSS triple (a, b, c): a counts
   id selectors, b counts class selectors and pseudo-classes, c counts type
   selectors; `*` and combinators count nothing. Triples compare
   lexicographically (each count saturates at 255). `:hover` therefore
   overrides a plain rule only when its selector is at least as specific,
   exactly as in CSS.
3. Inline `style=""` (beats every rule, as in CSS).

`!important` is not supported (§2.1), so there is no important layer.

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
| D8 | `overflow` is one value for both axes. | Simplicity. |
| D9 | *Removed:* width-only measurements no longer drop height information in subtrees that contain a column-wrap container (the only case where widths depend on heights). | |
| D10 | Measured leaves (text, images) are sized as fit-content: width `max(min-content, min(max-content, available))`, then the height is asked for at exactly that width (`Measurer.Height(w)`). Leaves never derive a width from a height (no aspect ratio yet). | Same as CSS for text; images get their intrinsic size. |
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
* D7 (cascade by source order and a pseudo-class layer instead of
  specificity): replaced by real specificity (§3).

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
5. **Glyphs**: the baseline rounds to the nearest device pixel, the pen x
   to the nearest quarter device pixel (§7.4).
6. **Images** fill their snapped content rectangle (§6.5).

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

### 6.6 Incremental repainting

Frames are byte-identical to a full repaint; only the work differs:

* **Damage regions**: each frame's display list is compared with the
  previous one. With the same structure, every operation that differs
  contributes its old and new device bounds (clipped); a changed clip
  contributes both clip rectangles; a changed group opacity the extent of
  the group. If operations were added or removed, the common prefix and
  suffix are identical and only the two middles are damaged, provided
  they leave the clip and group state the same; otherwise the whole frame
  is repainted. Rectangles are merged (at most 8). The new list is then
  replayed with each damaged rectangle as the base clip. This is exact
  because every raster operation is pixel-local under a clip (§6.1-6.5;
  shadow masks are computed over their whole extent). If more than 60%
  of the frame is damaged, it is repainted whole.
* Only the damaged rectangles are copied to the platform's buffer when
  it passes the same image again (platforms must not modify it between
  frames).
* **Incremental restyle**: a pseudo-class change restyles only the
  elements whose state changed (and their subtrees). If only paint
  properties change (colors, backgrounds, borders' color, radii,
  opacity, shadow, z-index), the boxes keep their layout; otherwise the
  document is rebuilt and laid out.
* `FLEXUI_CHECK_DAMAGE=1` makes every incremental frame and restyle also
  run in full and panic on any difference; all app and demo tests run
  with it (including 900 random interaction events).

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

### 6.5 Images

Each device pixel of the snapped destination rectangle samples the
source at its center with bilinear filtering in 16.16 fixed point on
premultiplied RGBA (edge texels clamp; no mipmaps, so strong
downscaling aliases), and composites source-over (§6.3) through the
clip. A 1:1 mapping copies source pixels exactly.

## 7. Text (M4)

### 7.1 Inline formatting and anonymous boxes

* An element is **inline** when its computed `display` is `inline` and
  every element inside it is inline too; otherwise it is laid out as a
  flex item (with a warning if it was meant to be inline).
* Among an element's children, each maximal run of text nodes and inline
  elements forms one **anonymous box** holding an inline formatting
  context: a flex item of the parent, with the inherited properties of
  the parent (`color`, `font-*`, `line-height`, `text-align`,
  `white-space`) and initial values for all others (`flex: 0 1 auto`,
  `min-width: auto`, no padding or border), exactly CSS's anonymous block.
  A run without any text after white-space processing is dropped, and its
  inline elements become flex items.
* Inside, text flows across inline elements: each text node is a run
  with its element's `font-weight`, `font-size`, `line-height` and
  `color`; white space collapses across run boundaries; shaping
  continues across element boundaries when the font is the same (so
  kerning applies, as in Chromium); widths are rounded per run.
* **Line boxes** (CSS 2.1 §10.8, all baselines aligned): each line's
  height is the maximum extent above and below the baseline of the
  block's strut and of every inline box with content on the line,
  including the boxes that enclose that content. An inline box extends
  above the baseline by its ascent plus half its leading (rounded down to
  a whole pixel), and below by the rest of its line height.
* Inline elements support `color`, `font-*`, `line-height` and
  `background-color` (painted behind each fragment's content area: the
  text's horizontal extent, from ascent to descent of the element's font,
  CSS 2.1 §10.6.1). Box properties (`padding`, `margin`, `border`,
  `width`/`height`, `position`, `overflow`, `opacity`, `box-shadow`,
  `border-radius`) warn and have no effect. `white-space`, `text-align`
  and `text-overflow` come from the block. There is no `vertical-align`
  (everything is on the baseline).
* Inline elements are hit-tested by their fragments, take `:hover` /
  `:active` / `:focus`, receive events, and report their bounds
  (`App.Bounds`) as the union of their fragments.
* **D24**: if the parent has `text-overflow: ellipsis` and `overflow`
  other than `visible`, the anonymous box takes that `overflow` and
  `text-overflow` (so `<div class=title>Long title</div>` ellipsizes). In
  CSS these properties do not inherit, so text directly inside a flex
  container never shows an ellipsis.
* The box is a leaf with a `layout.Measurer`: `Widths()` returns the
  min-content and max-content widths, and `Height(w)` lays the text out
  at content width `w` and returns the sum of the line heights. The
  engine sizes it as fit-content (D10) and asks for the height at exactly
  the final width.
* Painting: anonymous boxes are not positioned, so a positioned element's
  text paints right after its own background and border, before its
  element children (CSS 2 Appendix E, §6.4). An ellipsizing box clips its
  glyphs to its border box.

### 7.2 White space, breaking and lines

* `white-space: normal | nowrap`: tabs, newlines and spaces collapse to
  one space; spaces at the start and end of the run are removed.
  `pre | pre-wrap`: everything is preserved; `\r\n` and `\r` become
  `\n`; a final newline does not start an empty line. Tabs advance to the
  next multiple of 8 space widths from the start of the line.
* Break opportunities: Unicode line breaking (UAX #14, via
  `go-text/typesetting/segmenter`), except between two printable ASCII
  characters, where Chromium uses its own table instead of ICU
  (TextBreakIterator's ShouldBreakAfter): flexui uses the same table,
  measured from Chromium 153 for all 8836 pairs (`internal/text/asciibreak.go`;
  e.g. no break in `and/or` or `world!x`, a break in `"x"[y]`), and
  Chromium's rule that `-` before a digit breaks only after a letter or
  digit (`ABCD-1234`, not `-1`). Newlines are forced breaks with `pre`
  and `pre-wrap`. `nowrap` and `pre` never
  wrap. A word wider than the line overflows (`overflow-wrap: normal`).
* Lines are filled greedily: the next word is added if the line, without
  its trailing spaces, still fits the content width. The fit is measured
  as Chromium's line breaker does: the pieces before the last one with
  their rounded-up widths, the last piece with its exact advance rounded
  to the nearest 1/64 px (so CJK text whose advances are 15.0001 px at
  15 px fits exactly ten characters in 150 px).
* Spaces at the end of a line hang (do not count for fitting, alignment or
  intrinsic sizes), except with `pre` (they count) and with `pre-wrap`
  before a forced break or the end of the run, where they only
  **conditionally hang**: they count when they fit (CSS Text 3 §4.1.3).
* `text-align` offsets each line within the box's content width; centering
  rounds down to 1/64 px. Lines wider than the box start at its left
  edge. With `center` or `right`, a line that ends at a soft wrap after
  spaces has them removed and its last piece shaped again without them
  (Chromium's RemoveTrailingCollapsibleSpace), which drops kerning against
  the space ("Hello," "x."). If the line then no longer fits, its last
  word moves to the next line; a single word keeps its kerned width.
* `text-overflow: ellipsis` (with D24): a line wider than the box is cut at
  the last character boundary for which the content plus `…` (U+2026)
  fits, and `…` is drawn after it.

### 7.3 Fonts, shaping and metrics

* One family is built in: **Inter 4.1** Regular and Bold (static TrueType,
  SIL Open Font License, `internal/text/fonts/OFL.txt`). System fonts are
  never read. Every `font-family` value uses it (no warning: app CSS often
  lists stacks). `font-weight` above 500 selects Bold, otherwise Regular,
  which is CSS font matching for faces at 400 and 700. No synthetic bold
  or italics.
* **Font packs** (opt-in Go packages, `fonts/README.md`) add fallback
  faces: `fonts/tamil` (Noto Sans Tamil), `fonts/devanagari` (Noto Sans
  Devanagari), `fonts/arabic` (Noto Sans Arabic), `fonts/hebrew` (Noto
  Sans Hebrew), `fonts/cjk` (Noto Sans SC: Chinese, Japanese kana; a
  Regular face only, used for bold too; no Korean Hangul). A pack
  without a Bold face uses its Regular face for every weight. Importing one registers it; the fallback chain is Inter,
  then the registered packs in a fixed priority order (Tamil,
  Devanagari, ...), independent of import order. Packs must be
  registered before the first layout (imports do that).
* **Itemization** (UAX #24, as Chromium's ScriptRunIterator): the text
  is split into items of one script; characters of the Common and
  Inherited scripts (spaces, digits, punctuation, marks, joiners) belong
  to the item they are in, at the start of a line to the first strong
  script after them, and a closing paired bracket (Bidi_Paired_Bracket)
  to the item of its opening bracket. An item is also split at style
  (size, weight, language) changes, tabs and forced breaks.
* **Font fallback** (as Chromium's HarfBuzzShaper): an item is shaped
  with the first face of the chain; each maximal range of clusters that
  contain a .notdef glyph is shaped again with the next face, and so on.
  The last face keeps its .notdef glyphs (drawn as boxes).
* Shaping: HarfBuzz (go-text port) with default features (kerning,
  ligatures, contextual alternates), the item's script, and the content
  language from the inherited `lang` attribute (`en` without one), which
  selects language forms (e.g. Marathi and Nepali letter shapes in
  Devanagari). Right-to-left and mixed text: §7.6.
* Advances are kept in 1/65536 px, scaled as Chromium does on Linux:
  the face's advance as FreeType computes it (x_scale = FT_DivFix(size in
  26.6, unitsPerEm); advance = FT_MulDiv(units, x_scale, 64)), plus GPOS
  adjustments and offsets as HarfBuzz scales them ((v × ((size in 16.16
  << 16) / unitsPerEm) + 32768) >> 16). For Inter at whole-pixel sizes
  this is exact.
* Widths (matching Chromium to 1/64 px): a line's width is the sum, over
  its pieces (the text of one element between tabs and forced breaks,
  Chromium's inline items; script changes do not split them), of the
  piece's advance rounded **up** to 1/64 px. For min-content, a word's
  width is measured as Chromium does: both ends are rounded up to 1/64 px
  in the coordinates of its hard line, and the width is their difference.
* `font-size` is a length in 1/64 px, so sizes like `13.3px` are
  quantized (to 13.296875px).
* `line-height: normal` is round(ascent) + round(descent) + round(lineGap)
  in whole pixels, from the `hhea` metrics at the font size (Inter: 0.96875
  and 0.2412 em, no gap); a number is number × font-size rounded down to
  1/64 px; a length is used as is. The baseline is the line top plus
  round(ascent) plus half the leading, the leading being line height −
  (round(ascent) + round(descent)) and half of it rounded down to a whole
  pixel. With `line-height: normal`, every fallback face used on a line
  also takes part with its own rounded ascent, descent and line height
  (as in Chromium: a Tamil word makes a 16px line 22px tall, Devanagari
  23px); with any other line-height only the primary face counts.

### 7.4 Glyph rendering

* Glyph outlines are rasterized by flexui (`internal/text/mask.go`) with
  integer math only: outline points in half font units (exact), scaled to
  1/256 device px; curves flattened to under 1/16 px error; coverage from
  16 sample rows per pixel with exact horizontal span coverage, scaled to
  0..255 with rounding. No hinting. Grayscale anti-aliasing only.
* Placement (§6.1 rule 5): the glyph origin's x is rounded to the nearest
  quarter device pixel (four cached variants per glyph), the baseline to
  the nearest device pixel. The font size scales exactly with the device
  scale; layout does not change with the scale.
* The mask is composited with the text color as coverage (§6.3), so
  `color` alpha and clips apply as for any fill.

### 7.5 Text deviations

| # | Deviation | Reason |
|---|---|---|
| D21 | Glyph anti-aliasing differs from Chromium (FreeType coverage with Skia's luminance-dependent contrast/gamma "preblend"). Glyph positions agree; individual edge pixels differ, by up to about 50 of 255 levels, mean error about 0.3-0.5 levels per text document. | Integer-only own rasterizer; deterministic everywhere. |
| D22 | Layout is independent of the device scale. Chromium rounds `line-height: normal` metrics in device pixels, so at 2x its lines can be up to 0.5 px (one device pixel) shorter. | Same layout on every display; HiDPI is a paint-time scale only. |
| D23 | Shaping at a soft line end. Chromium removes the trailing spaces of a soft-wrapped line that is not aligned to its start side and shapes the end of the line again without them, which drops kerning against the space (e.g. "Hello," or "." followed by a space): this is emulated (§7.2), with Chromium's moving the last word down when the line no longer fits; the line-fit rule of §7.2 (last piece unrounded) fixed one more case. 2 random Latin documents (within 0.5 px) remain, both with `text-overflow: ellipsis` / nowrap or flex items whose width is the text's own; we could not derive those. | Rare; under 0.5 px. |
| D24 | Text in an element with `text-overflow: ellipsis` and clipping `overflow` ellipsizes (§7.1); in CSS it would not, because the anonymous box does not inherit them. | What authors expect; Chromium comparisons wrap text runs in an equivalent block that takes these properties. |
| D30 | `em` and `i` are not italic, and `font-style` is not supported: only upright Regular and Bold faces are bundled. | Binary size; add an italic face later. |
| D32 | Float rounding of text widths: Chromium adds glyph advances as float32 (per shaping run, then per item) before rounding a piece up to 1/64 px. When the exact width is within float32 precision (a few 1/65536 px) above a 1/64 px boundary, Chromium's width is 1/64 px smaller. Only fonts whose scaled advances are not exact in float32 are affected (the Noto packs, units per em 1000); Inter at whole-pixel sizes is exact. Seen in 1 of 150 random multi-script documents. | Emulating float32 summation in Chromium's run order costs more than a 1/64 px difference is worth. |
| D33 | Bidi residuals not derived: in 7 of 150 random bidi documents, 2 differ by 1/64 px (D32), 2 by about 0.4 px in a line ending in kerned punctuation before a space (the trailing-space reshaping of §7.2 in mixed-direction lines: the same set-up keeps the kerning in one document and drops it in another), 2 have a line 1-2 px taller in Chromium (lines with isolates), and 1 breaks a line at a different place. | The remaining cases did not reduce to a rule after several attempts; all other bidi documents match glyph for glyph. |
| D34 | CJK punctuation spacing: Chromium trims the spacing of full-width punctuation by default (`text-spacing-trim: normal`: adjacent pairs such as "。「" collapse, an opening bracket at a line start and a closing one at a line end when the line would not fit otherwise become half-width, applied with the font's `halt` feature by language-dependent rules). flexui keeps full-width punctuation, so CJK text with such punctuation is wider and can break elsewhere (`testdata/text/docs/script_cjk_punct.html`). Chinese and Japanese text without such punctuation pairs matches Chromium exactly. | Blink's HanKerning rules are language-dependent; left for later. |

### 7.6 Bidirectional text

* **Direction**: the `direction` property (inherited) and the `dir`
  attribute (`ltr`, `rtl`, `auto`; a presentational hint, overridden by
  author CSS). `dir="auto"` is resolved as HTML does: the first strong
  character in the element's text (skipping descendants with their own
  `dir`, and digits), `ltr` without one. A block's direction is the base
  direction of its paragraphs and resolves `text-align: start/end`.
* **Isolates**: an inline element with a `dir` attribute is a bidi
  isolate (browsers: `unicode-bidi: isolate`), as if its text were
  wrapped in LRI or RLI ... PDI. `direction` without `dir` on an inline
  element has no effect on reordering (`unicode-bidi: normal`), as in
  browsers. `unicode-bidi`, `<bdi>` and `<bdo>` are not supported.
* **Algorithm**: UAX #9 (go-text's implementation) per hard line (a
  forced break separates paragraphs), with the block's direction as the
  paragraph direction. Items and width pieces are split at level
  changes; odd levels are shaped right to left. Each line is reordered
  with rules L1 (trailing white space at the paragraph level) and L2;
  isolate controls take part as zero-width separators, as in Chromium.
* **Lines**: alignment as in §7.2 with `start` on the right in
  right-to-left paragraphs; a line wider than its box overflows on the
  end side (to the left in right-to-left text). The trailing-space
  reshaping of §7.2 applies to lines not aligned to their start side.
  The ellipsis (D24) is drawn at the end side.
* **Flex layout** in a right-to-left container: a row's main axis runs
  right to left (`row-reverse` left to right); a column's cross axis
  starts at the right (`align-items: flex-start` is the right edge;
  `wrap-reverse` flips it back). The overflow fallback of
  `justify-content: space-around/evenly` is the writing-mode start (the
  right). Absolutely positioned boxes with `left`, `right` and `width`
  ignore `left`; relative positioning with both `left` and `right` uses
  `right` (CSS 2 §9.4.3, §10.3.7).
* Verified against Chromium: hand-written documents and 150 seeded random
  bidi documents (every grapheme's visual position compared), 400 random
  right-to-left flex layout trees. Exceptions: D32, D33.

## 8. Interaction (M5, M6)

### 8.1 Handlers and events

* An element declares handlers by name with attributes: `onclick="Save"`,
  `onkeydown`, `onfocus`, `onblur` (and `oninput`, `onchange` for inputs,
  M8). Go registers them with
  `app.Handle("Save", func(c *flexui.Ctx) {...})`. Other `on*` attributes
  warn. A name with no registered Go handler warns once (in
  `App.Warnings`) when it would have run.
* **Propagation**: an event goes to its target element, then bubbles to
  each ancestor; every element on the way with a matching attribute has
  its handler called, deepest first, until one calls
  `Ctx.StopPropagation()`. There is no capture phase and no default
  action to prevent (except as listed for keys in M6).
* **click**: a primary-button press and release form a click. Its target
  is the nearest common ancestor of the elements under the press and the
  release (as in browsers), so pressing one button and releasing on
  another clicks neither. Text belongs to its parent element.
* `Ctx` gives the event type, the `id` of the element whose handler runs
  (`ID`) and of the target (`TargetID`), the pointer position or key, and
  `Attr(name)` to read attributes such as `data-id` from the element whose
  handler runs.

### 8.2 State and re-rendering

* Go owns the state. A view is either static (`SetView`) or a function of
  state (`SetViewFunc`), usually an `html/template`.
* After any handler runs (or after `App.Invalidate()`), the view function
  is called again before the next event or frame, and its HTML replaces
  the document. The new document is built from scratch (no tree diffing);
  interaction state that must survive (focus, scroll positions, M6) is
  carried over by element `id`, or by tree position for elements without
  an id.
* **Caching**: each frame recomputes only the stages whose inputs
  changed: parse (only when the view's HTML string changed), style
  (HTML, CSS or pseudo-class state changed; paint-only pseudo-class
  changes restyle in place, §6.6), layout (style or viewport changed),
  placement (scrolling), paint (only damaged rectangles, §6.6). Parsed style sheets and shaped
  text runs are cached by source and parameters across rebuilds, so a
  re-render after a handler re-shapes only text that changed. An
  unchanged frame copies the retained image. A frame produced
  incrementally is always byte-identical to a fresh render of the same
  HTML (tested).

### 8.3 Hit testing

* The element under a point is the **last element in paint order**
  (§6.4: z-index, stacking contexts, tree order) whose border box contains
  the point and which is not clipped there by an ancestor's `overflow`
  (padding-box rectangle). Painting and hit testing use one traversal, so
  they cannot disagree; `TestHitTestMatchesPaint` checks this pixel by
  pixel on random overlapping documents.
* Corner radii are ignored for hit testing (the border box rectangle
  counts). Elements with `opacity: 0` or a transparent background are still
  hit, as in browsers. Text hits its parent element. A box with negative
  `z-index` whose stacking context is the canvas is under `body`, so
  `body` is hit there (as in browsers).

### 8.4 Pseudo-classes and focus

* `:hover` applies to the element under the pointer and all its
  ancestors; it is recomputed when the pointer moves, when content
  scrolls or is re-laid-out under it, and cleared when the pointer leaves
  the surface. `:active` applies to the pressed element and its ancestors
  from press to release (or until a drag scroll starts), and to a button
  held with Space.
* **Focusable** elements: `button`, `a`, `input`, and any element with a
  `tabindex` attribute, unless inside a `display: none` subtree. `:focus`
  applies to the focused element only (no `:focus-within` /
  `:focus-visible`). Pressing the pointer focuses the nearest focusable
  element at or above the target; pressing elsewhere blurs.
  `App.Focus(id)` focuses programmatically.
* **Keyboard**: key events go to the focused element (or `body`) and
  bubble (`onkeydown`, with `Ctx.Key` and `Ctx.Mods`). Unless a handler
  calls `Ctx.PreventDefault()`: Tab / Shift+Tab move focus in sequential
  focus order (positive `tabindex` ascending, then `tabindex="0"` and
  naturally focusable elements in tree order; negative `tabindex` is
  skipped; wraps around); Enter clicks a focused `button` or `a`; Space
  clicks a focused `button` on release.
* **D26**: there is no general focus ring: the default style sheet gives
  `button` and `input` a blue border on `:focus`; other focusable elements
  need a `:focus` rule in the app's CSS.
* **`disabled`** (on `button` and `input`): the element matches
  `:disabled`, is not focusable, gets no `:active`, and no click is
  dispatched to it or, through it, to its ancestors (as in browsers).
* **Focus events**: `onfocus` and `onblur` run when focus changes by
  pointer, keyboard or `App.Focus`. They do not bubble. They do not run
  when a re-render merely carries focus over to the new document.

### 8.5 Scroll containers

* Scroll containers that overflow draw **overlay scrollbars**: a thumb
  per scrollable axis, 6px wide, 2px from the padding box edges, at
  least 20px long, rounded, black at 35% opacity, over the content. The
  thumb can be dragged (pressing it does not focus, activate or click).
* `overflow: scroll` makes an element a scroll container that the user
  can scroll; `overflow: hidden` only clips (as in CSS). The scroll range
  is the content extent (layout's scrollable overflow, measured from the
  padding box) minus the padding box size.
* **Wheel**: scrolls the innermost scroll container under the pointer;
  what it cannot absorb passes to the next one outward (scroll chaining).
  Deltas are logical pixels; positive scrolls down/right.
* **Drag**: pressing over a scroll container and moving more than 4
  logical px (in either axis) scrolls it with the pointer, like touch.
  Such a press does not click, and `:active` ends when the drag starts.
* Scrolling moves the descendants' boxes (painting, clipping and hit
  testing all follow) without re-running layout. Offsets are clamped
  after every layout.
* Scroll offsets are kept across re-renders by element key (§8.2: `id`,
  or tree position), and re-clamped to the new content.
* **D25**: scrollbars are overlay scrollbars and take no space (like
  Chromium with hidden scrollbars, as in the oracle); they are always
  shown while content overflows. There is no keyboard scrolling, no track
  clicking and no inertia yet.

### 8.6 Text inputs (M8a)

* An `<input>` holds an editable single line (`internal/edit`): text,
  caret and selection anchor, on grapheme cluster boundaries. Newlines
  and control characters are dropped from inserted text; tabs become
  spaces.
* **Value**: the `value` attribute sets the text when the input first
  appears and whenever the attribute changes between renders (if it
  differs from the current text); otherwise the user's text stays. So
  both patterns work: a controlled input (`value="{{.Name}}"` plus an
  `oninput` handler storing `Ctx.Value`) keeps its caret while typing,
  and an uncontrolled input keeps its edits across re-renders.
  `App.Value(id)` reads the current text. State is kept by element key
  (§8.2).
* **Typing**: platforms send committed text (`EventText`;
  `headless.Session.Type`). Keys, after `onkeydown` handlers (which can
  `PreventDefault`):

  | Key | Action (Shift extends the selection) |
  |---|---|
  | ArrowLeft / ArrowRight | one grapheme; with Alt or Ctrl one word; with Meta to the line start/end. Logical movement: Right is forward in a left-to-right input, backward in a right-to-left one (as Chromium; not visual movement through mixed text) |
  | Home / End, ArrowUp / ArrowDown | to the start / end |
  | Backspace | the selection, else one **code point** of the cluster before the caret (a vowel sign, an accent), or the whole cluster for emoji sequences (ZWJ, flags, skin tones, keycaps), as Chromium; Alt/Ctrl: word, Meta: to the start |
  | Delete | the selection, else one grapheme cluster (Alt/Ctrl: word, Meta: to the end) |
  | Ctrl/Meta + A, C, X, V | select all, copy, cut, paste (`App.SetClipboard`; in-memory by default) |
  | Enter | `change` event if the value changed |
  | Tab | focus traversal (§8.4) |

  Words: a word is a run of letters, digits, `_` and `'`; moving left goes
  to the start of the previous word, right to the end of the next one
  (macOS behavior, also used elsewhere).
* **Pointer**: pressing in a focusable input focuses it and puts the caret
  at the nearest grapheme boundary; dragging extends the selection (the
  text does not autoscroll while dragging).
* **Events**: `oninput` after every change of the text (typing, deletion,
  cut, paste) and `onchange` when the input loses focus or Enter is
  pressed, if the value differs from the last `change` or from when it
  was focused (as in browsers, `change` comes before `blur`). Both bubble;
  `Ctx.Value` holds the text.
* **Drawing**: the text (or the `placeholder` attribute, gray, when
  empty) on one line, vertically centered and clipped to the content box,
  scrolled horizontally so the caret stays visible. While focused, a
  1px caret in the text color (or a light blue selection highlight).
* **Direction**: an input with `dir="rtl"` (or `direction: rtl`) shows
  its text right-aligned; text is bidi-reordered (§7.6) in every input;
  a selection across a direction change is drawn as several rectangles;
  the caret at the logical start or end of the text is drawn at the
  line's start or end side. Clusters and Backspace behavior were checked
  against Chromium for Devanagari, Tamil, Arabic, Hebrew, combining
  accents and emoji (`internal/edit` TestChromiumEditing).
* Not supported yet: undo, double-click
  word selection, autoscroll while drag-selecting, `maxlength`,
  `readonly`, password fields. **D31**: the caret does not blink (frames
  are deterministic; a platform may add blinking later).


### 8.7 IME composition

Measured in Chromium 153 (`<input>`, CDP `Input.imeSetComposition` and
`insertText`) and matched:

* Platforms send `EventCompose` (marked text, caret within it),
  `EventComposeCommit` (final text) and `EventComposeCancel`
  (`headless.Session.Compose`, `Commit`, `CancelComposition`).
* The marked text is part of the input's value while composing (it
  replaces the selection when the composition starts); every update
  sends an `input` event with `Ctx.Composing` set. The commit replaces
  the marked text and sends an `input` event (Chromium's last one has
  `isComposing` set and is followed by `compositionend`; flexui has no
  composition events for handlers). A cancel removes the marked text.
* Focus moving away or a pointer press commits the marked text as it is.
* The marked text is drawn with a 1px underline in the text color.
* `App.CaretRect()` (`platform.CaretRecter`) gives the focused input's
  caret rectangle in logical pixels, for the platform's candidate window.
* Platforms: Windows through IMM32 (`platform/windows/ime_windows.go`),
  macOS through NSTextInputClient (`platform/darwin/ime_darwin.go`, with
  assembly for the struct-returning methods). Both are compile-checked
  only; stage 9 on each OS verifies them (READMEs list the risks).
* Not supported: per-clause underlines (thick/thin) and the IME's own
  selection highlight within the marked text; `compositionstart` /
  `compositionend` handlers.
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
  `testdata/render/docs` (17 documents, 4 also at 2x) are
  screenshotted by `scripts/screenshot.cjs` (`make crosscheck`) with a
  base sheet encoding D1-D4 and D20. For each one the test reports the max
  error, the mean absolute channel error, and the share of pixels off by
  more than 8 and 32 levels. The default bound is near-exact: mean ≤ 0.05
  and no pixel off by more than 32. Documents above that are listed with
  their deviation (D17 shadows, D18 curve AA, D19 opacity rounding) and
  bounds about 1.5x their measured error. Current numbers are in
  `spec/STATUS.md`.
* Text layout is compared with Chromium **exactly** (1/64 px): the border
  box of every element and anonymous text box, and every line's text, x,
  y, width and height (`internal/render/text_test.go`). Chromium loads the
  bundled fonts through `@font-face`, and every text run is wrapped in an
  `<x-t>` block equivalent to the anonymous box (`scripts/flexui-page.cjs`,
  `scripts/textcheck.cjs`). Inputs: 12 hand-written documents in
  `testdata/text/docs` (wrapping, sizes, weights, alignment, line height,
  white space, ellipsis, row items, nested content-sized boxes, mixed
  content, inline formatting) and 300 seeded random documents, plus 150
  seeded random documents of inline content (nested spans with sizes,
  weights, line heights, links, `br`, alignment). The oracle runs in
  standards mode with flexui's default style sheet. Accepted mismatches are listed
  in `knownTextDiffs` with a deviation and a tolerance (D14 rounding of
  flex sizes: 2/64 px; D23: 0.5 px); the test fails if a listed case
  starts matching exactly. Text RenderHTML documents are also in the
  screenshot diff, with bounds for D21/D22.
* **Determinism gate** (M7): `TestHashes` renders every golden document
  (and 2x variant) and compares SHA-256 hashes of the pixels with
  `testdata/render/hashes.txt`. `scripts/check-portability.sh` runs it on
  linux/386 besides the normal amd64 run, builds and vets everything
  (including the macOS backend and its stages) for linux, darwin and
  windows on amd64/arm64 with `CGO_ENABLED=0`, and checks the dependency
  rules for every target (no platform backend, purego or x/sys in the
  core; purego only in `platform/darwin`). On macOS and Windows, run
  `go test ./internal/render -run TestHashes` there. App goldens
  (`testdata/app`) are compared byte for byte too.
* `go test ./internal/layout -run TestReduce -v -args -reduce=CASE`
  minimizes a failing case against a live Chromium
  (`scripts/chrome-serve.cjs`). It deletes subtrees and declarations while
  the mismatch persists, which turns a random tree into a 2-4 node
  reproduction.
