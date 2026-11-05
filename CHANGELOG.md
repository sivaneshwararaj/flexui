# Changelog

All notable changes are recorded here. The project uses semantic
versioning with `v0.x` tags until the API settles.

## Unreleased

### Added
- IME composition model: `platform.EventCompose`, `EventComposeCommit`,
  `EventComposeCancel`, `platform.CaretRecter` / `App.CaretRect`,
  `Ctx.Composing`, headless `Compose`/`Commit`/`CancelComposition`;
  marked text in the value and underlined, as in Chromium.
- Inputs: right-to-left inputs (`dir="rtl"`: right-aligned, arrow keys
  reversed as in Chromium), bidi-reordered text with multi-rectangle
  selections in every input.
- Bidirectional text: `direction`, the `dir` attribute (`ltr`, `rtl`,
  `auto`; isolates on inline elements), `text-align: start | end` (start
  is the new initial value, same as `left` in left-to-right text), UAX #9
  levels and line reordering, right-to-left flex layout (row axes,
  column cross axes, absolute and relative positioning), font packs
  `fonts/arabic` and `fonts/hebrew`. Checked against Chromium: 150
  random bidi documents glyph by glyph, 400 random right-to-left layout
  trees (deviations D32, D33).
- Multi-script text: itemization by script (UAX #24, paired brackets),
  Chromium-style font fallback per cluster, the `lang` attribute as the
  shaping language, and opt-in font packs `fonts/tamil` (Noto Sans
  Tamil) and `fonts/devanagari` (Noto Sans Devanagari). Checked against
  Chromium (the packs loaded through `@font-face`) on hand-written and
  150 seeded random multi-script documents (`testdata/text/script.json`).
  `testdata/multilingual` pages for checking by eye.
- Multilingual paste/copy tests (`internal/testtext` samples), Windows
  UTF-16 clipboard conversion tests, and stage 8 on macOS and Windows: a
  real-clipboard round trip of the same strings.

### Changed
- Inputs: Backspace deletes one code point of a cluster (vowel signs,
  accents), whole clusters only for emoji sequences, as Chromium; Delete
  still deletes a whole cluster.
- Chromium allows a line break after every run of spaces (also before
  `!`, `)` ...); flexui now does too.
- Glyph advances are scaled as Chromium does on Linux (FreeType
  advances, HarfBuzz GPOS scaling, 1/65536 px); for Inter at whole-pixel
  sizes nothing changes.
- Line breaks between two printable ASCII characters follow Chromium's
  own table (measured from Chromium 153 for all pairs) instead of UAX #14
  plus the `/` and `!` tailoring.
- Centered and right-aligned lines that end at a soft wrap drop the
  kerning against the removed trailing space, as Chromium does (fixes 3
  of the 6 cases of deviation D23).
- `line-height: normal` takes the metrics of fallback fonts used on a
  line into account.

### Fixed
- Inputs: word movement and word deletion (Ctrl/Alt+Arrow,
  Ctrl/Alt+Backspace) treated vowel signs, viramas, combining marks and
  joiners as word breaks; words are now judged per grapheme cluster.
- Windows clipboard read is bounded by the memory block size
  (`GlobalSize`) instead of scanning for a NUL.

## v0.1.0 (2026-11-05)

First tagged state. All seven platform verification stages pass on macOS
(Version 27.2) and Windows (build 26300, 125% scaling); the demo runs on
both. Module path: `github.com/sivaneshwararaj/flexui`.

### Fixed
- Windows clipboard: `OpenClipboard` is retried for about 100 ms (another
  process can hold the clipboard briefly), `EmptyClipboard` is checked,
  and failures are logged with `-debug`.

### Added
- Headless test of Ctrl+C/X/V (the copy test used Cmd only); Windows
  stage 6 prints a `RESULT` line for every clipboard read and write.
- Documentation for users and coding agents: `README.md`,
  `docs/GUIDE.md`, `AGENTS.md`, `llms.txt`; Go examples
  (`example_test.go`) checked by `go test`. `Ctx.InputValue(id)` reads an
  input inside handlers.
- `platform/windows` (x/sys/windows, no cgo) with `windows.Run`, staged
  verification commands and README (stages 1-7 pass on Windows);
  `-window` on Windows for the counter and demo examples. Dependency:
  `golang.org/x/sys` v0.41.0 (Windows backend only).
- Damage-region repainting and copying, incremental paint-only restyle
  (`FLEXUI_CHECK_DAMAGE=1` verifies them against full repaints); demo
  frame benchmarks at 2560x1600.
- M9: `examples/demo`, a contacts manager (list, form, modal) using only
  the public API, with a scripted headless session and goldens;
  `Ctx.Focus` (focus after the re-render).
- M7: portability gate (builds/vets all targets, dependency rules on every
  OS, render hashes on linux/386, `TestHashes` with
  `testdata/render/hashes.txt`); `platform/darwin` (purego) with
  `darwin.Run`, staged verification commands and README (stages 1-7
  pass on a Mac); `examples/counter -window`. New dependency:
  `github.com/ebitengine/purego` v0.10.0 (macOS backend only).
- M8a text inputs: editing model (`internal/edit`), keyboard editing and
  movement, clipboard (`App.SetClipboard`), pointer caret and selection,
  placeholder, `oninput`/`onchange`, `Ctx.Value`, `App.Value`,
  `platform.EventText` handling, `headless.Session.Type` and `Clipboard`.
- M5b: inline formatting across `span`, `a`, `strong`/`b`, `em`/`i`,
  `br` (mixed sizes, weights, line heights, colors, backgrounds); `<img>`
  (PNG/JPEG, `data:` URLs or `Options.Assets`); a default style sheet for
  links, buttons and inputs; `:disabled` and the `disabled` attribute;
  `onfocus`/`onblur`; overlay scrollbars with draggable thumbs;
  `display: inline`; `layout.AspectRatio`.
- M6 interaction: hit testing in paint order, `:hover`/`:active`/`:focus`
  styles, pointer and keyboard focus (Tab order with tabindex), Enter and
  Space activation, `onkeydown` with `Ctx.PreventDefault`, `App.Focus`,
  `App.FocusedID`, scroll containers with wheel (chaining) and drag
  scrolling, `App.ScrollOffset`, focus and scroll kept across
  re-renders; `platform.EventPointerLeave`; `headless.Session.Drag`,
  `Leave`.
- M5 handlers and state: `onclick` handlers bound with `App.Handle`,
  bubbling and `Ctx.StopPropagation`, `Ctx.Attr`/`ID`/`TargetID`,
  `App.Invalidate`, `App.Bounds`, re-render after handlers with staged
  caching (parse/style/layout/place/paint, sheet and text caches,
  retained frames); `headless.Session` for stepping synthetic events;
  `platform.Mod*` constants; `examples/counter`.
- M4 text: bundled Inter 4.1 (OFL), shaping, line breaking, wrapping,
  `white-space`, `text-align`, `line-height`, `text-overflow: ellipsis`,
  integer-only glyph rasterizer; text is measured in layout through the
  new `layout.Measurer` (`Widths`, `Height(w)`). Text layout is compared
  exactly against Chromium (bundled fonts loaded via `@font-face`) on
  hand-written and 300 random documents. New dependencies:
  `github.com/go-text/typesetting` v0.3.5, `golang.org/x/image` v0.23.0.
- M3: `flexui.RenderHTML(html, css, w, h)` and `CheckHTML`; `App.Frame`
  renders views. HTML5 parsing (`golang.org/x/net/html` v0.38.0), style
  sheets, selectors and the cascade, inheritance, colors,
  `background-color`, `border`, `border-radius`, `opacity`, `box-shadow`,
  text properties (stored for M4), `z-index` stacking contexts and CSS
  paint order. Golden documents and a pixel diff against Chromium
  screenshots with max/mean error reporting.
- Public root package `flexui` (`New`, `Options`, `App.SetView`,
  `App.SetViewFunc`, `App.Handle`, `Ctx`), `platform` interfaces, and
  `platform/headless` with `Render`/`RenderImage`. Rendering of views
  arrives in M3; until then frames are blank.
- Internal flexbox layout engine verified against Chromium (M1).
- `examples/` module written against the public API only.

- Software rasterizer (M2): `Canvas` draw interface, display list,
  integer-only `Raster` with rounded rects, borders, box shadow, clipping,
  opacity groups and HiDPI scale; documented pixel-snapping rule; golden
  PNG tests.
- Randomized differential layout testing against Chromium (2400 seeded
  trees) and a test-case reducer.

### Changed
- `span`, `a`, `strong`, `b`, `em`, `i` are inline by default (they were
  flex items); buttons and inputs have default styles. Documents are laid
  out in standards mode (the Chromium oracle now uses a doctype).
- The cascade orders rules by CSS specificity (id, class/pseudo-class,
  type), then source order; the source-order cascade and its
  pseudo-class layer (deviation D7) are gone.
- Implementation packages moved under `internal/`.

### Fixed (layout, found by differential testing)
- Cyclic percentages: horizontal percentages, percentage padding/margins
  and gaps now resolve against the final container size.
- Intrinsic sizes of row containers (contributions, flex-basis handling),
  fit-content of wrapping containers, column-wrap line breaking and widths.
- §9.8 definiteness (stretched items, definite flex-basis), stretch only
  for `auto` cross sizes.
- Absolute positioning: auto margins, `align-self` with both vertical
  insets, shrink-to-fit from the static position.
- Distribution rounding and overflow fallbacks match Chromium, including
  reversed containers.
