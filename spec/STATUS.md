# Status

Last updated after step 6 of the multilingual work (bidirectional
text). Next steps: cluster-aware and bidi-aware editing in inputs (step
7); IME; CJK.

### Bidirectional text (step 6)

* `direction`, `dir` (incl. `auto` and isolates on inline elements),
  `text-align: start/end`, UAX #9 reordering, right-to-left flex layout,
  Arabic and Hebrew packs (+288 KB, +38 KB; the bidi code adds ~120 KB
  to every binary).
* Verified against Chromium: hand-written bidi documents and 150 random
  ones with every grapheme's position compared (142 exact; 8 recorded
  as D32/D33), 400 random right-to-left layout trees (all match except 6
  that differ the same way without direction: D11, D12, D14).
* Found on the way: Chromium breaks after every space run (fixed for all
  text); go-text's bidi Runs merge levels 0 and 2 (worked around by
  reading its per-character levels); go-text drops the text after a
  paragraph separator (levels are computed per hard line).
* Not done: inputs are left to right only (`dir` on `<input>` has no
  effect on its text yet; step 7); `unicode-bidi`, `<bdi>`, `<bdo>`;
  vertical text.

### Multi-script text (step 5)

* Itemization by script, Chromium-style per-cluster font fallback, the
  `lang` attribute as shaping language, opt-in packs `fonts/tamil`
  (+95 KB) and `fonts/devanagari` (+364 KB). The Tamil paste problem is
  fixed when the app imports `fonts/tamil` (review/step5: before/after).
* Verified against Chromium with the same fonts via `@font-face`: 3
  hand-written script documents and 150 seeded random multi-script
  documents (boxes, line breaks, line widths, heights) match exactly,
  except one random document 1/64 px off (D32, float32 rounding). All
  2,510 layout cases, the Latin text documents and the render goldens
  are unchanged.
* Found on the way and fixed (they also affect Latin text): Chromium's
  ASCII line-break table (all 8,836 printable pairs now match), the
  kerning dropped at the end of centered/right-aligned soft-wrapped
  lines (3 of 6 D23 cases fixed), FreeType/HarfBuzz advance scaling.
* Not verified: right-to-left scripts (step 6), scripts without a pack
  (drawn as .notdef boxes, as before), font packs on real macOS/Windows
  windows (stage 8 imports them; not run yet).
* `RenderHTML` (no App) does not draw `<input>` values; this predates
  the multilingual work (input text is drawn in apps).

### Tamil paste diagnosis (Phase 1)

| Layer | Finding |
|---|---|
| Clipboard | Correct. Headless: 13 multilingual strings paste and copy back byte-identical (`TestPasteMultilingual`). Windows UTF-16 conversion, surrogate pairs and NUL termination are unit-tested on every OS (`platform/windows/utf16_test.go`); the read is now bounded by `GlobalSize` (it scanned for a NUL without a limit). macOS passes Go strings to `stringWithUTF8String:` as NUL-terminated copies (checked in purego v0.10.0's `addValue`). The real-clipboard round trip is stage 8 on each OS: **not run yet**. |
| Buffer | Data intact; clusters follow UAX #29 (Unicode 15.1 in go-text v0.3.5, incl. Devanagari conjuncts as one cluster; Tamil splits at the pulli as UAX #29 specifies). **Bug fixed:** word movement and Ctrl/Alt+Backspace treated vowel signs, viramas and joiners as word breaks (Ctrl+Backspace at the end of `नमस्ते दुनिया` deleted both words). Remaining difference from browsers: Backspace deletes a whole cluster (`स्ते`), browsers delete one code point in Indic clusters (step 7). |
| Drawing | **At fault for what you saw.** Inter has no Tamil glyphs, and each code point is drawn as its own .notdef box (vowel signs and pulli included), so `தமிழ்` shows five boxes: looks garbled, data is not (`review/phase1`). Fixed by font fallback (step 5). |

| Milestone | State |
|---|---|
| M0 Spec and scaffolding | done |
| M1 Layout engine | done |
| M2 Rasterizer | done |
| M3 HTML+CSS to pixels | done (no images yet) |
| M4 Text | done |
| M5 Handlers and state | done |
| M6 Interaction | done |
| M5b Inline formatting, images, default styles, disabled, focus events, scrollbars | done |
| M7 Portability gate + macOS | gate done; macOS stages 1-7 passed on an Apple-silicon Mac (macOS 27.2), incl. typing and resizing; clipboard, scale changes between displays and Intel Macs unverified; stage 8 (multilingual clipboard) not run |
| M8 Text input | M8a done (editing model, headless); IME composition with the macOS step |
| Windows backend | stages 1-7 pass on Windows (build 26300), demo runs; clipboard both ways after an OpenClipboard retry; stage 8 (multilingual clipboard) not run (see platform/windows/README.md) |
| M9 Demo app | done headless (`examples/demo`); the window is untested |

## Works

* Public API (root package `flexui`): `RenderHTML(html, css, w, h)`,
  `CheckHTML` (warnings), `New`, `Options`, `App.SetView`,
  `App.SetViewFunc`, `App.Handle`, `App.Warnings`, `Ctx`; `App.Frame`
  renders the view, so `headless.Render(app, "out.png")` produces real
  pixels. Implementation lives under `internal/`. `examples/` is a separate
  module (`replace ../`) using only the public API. MIT `LICENSE`,
  `CHANGELOG.md`.
* `dom`: HTML5 parsing via `golang.org/x/net/html` (v0.38.0, Go 1.24
  compatible). `<body>` is the root; `<style>` elements are collected;
  unknown elements, `<script>` and non-text `<input>` types warn.
* `style`: style sheets (rules, selector lists, comments, at-rules skipped
  with a warning), selectors (type, `*`, `#id`, `.class`, descendant,
  child, `:hover`/`:focus`/`:active`), the cascade of spec §3
  (specificity, then source order, then inline), inheritance,
  `currentcolor`. All
  layout properties, and the paint and text properties of spec §2.1:
  colors (hex, `rgb()`, 148 named), `background(-color)`, `border`,
  `border-color`/`-style`/`-radius`, `opacity`, `box-shadow`, `font-*`,
  `line-height`, `text-align`, `white-space`, `text-overflow` (text ones
  parsed and inherited, used from M4). Border widths are snapped as
  computed values, as browsers do.
* `layout`: the full flexbox algorithm in `spec/SUBSET.md` §2.
  It is verified against Chromium with 110 hand-written cases plus 2400
  seeded random trees (`testdata/layout/chrome.json`, `make crosscheck`).
  All match to 1/64 px except 25 random cases (about 1%), which are
  listed with deviations D11-D14. A reducer (`TestReduce -reduce=CASE`)
  shrinks any mismatch to a minimal tree against a live Chromium.
  Intrinsic sizing is linear-ish (about 7 measure calls per leaf on a
  10-deep auto-sized tree).
* `paint`: `Canvas` interface, display list, integer-only rasterizer
  (rects, rounded rects, borders, box shadow, rect/rounded clips, opacity
  groups, HiDPI). Snapping and blending are specified bit-exactly in spec
  §6.1-6.3. Steady-state frames allocate nothing; a 320x200 card at 2x
  renders in about 2 ms.
* `text` (M4): bundled Inter 4.1 Regular/Bold (OFL), HarfBuzz shaping in
  font units, UAX #14 breaking (tailored to Chromium), greedy wrapping,
  `white-space` (normal, nowrap, pre, pre-wrap with conditional hanging),
  `text-align`, `line-height`, `text-overflow: ellipsis`, and an
  integer-only glyph rasterizer (quarter-pixel positioning, cached
  masks). Layout leaves use `layout.Measurer` (`Widths()`, `Height(w)`).
  Text layout matches Chromium exactly (1/64 px) on 12 hand-written and
  300 random documents: 2799 boxes and 2587 lines compared; 33 random
  documents differ by at most 2/64 px (flex rounding, D14) and 2 by under
  0.5 px (kerning at a line end, D23).
  * Works around a go-text v0.3.5 bug that skipped most class kerning
    (`internal/text/coverage.go`, guarded by `TestKerning`).
* Inline formatting (M5b): `span`, `a`, `strong`/`b`, `em`/`i`
  (upright), `br`; runs with their own size, weight, line height, color
  and background; line boxes with mixed metrics (CSS 2.1 §10.8); links
  hit-tested, hovered and focused by fragment. Text layout compared with
  Chromium exactly on 16 hand-written documents (incl. 4 inline), 300
  random block documents and 150 random inline documents: 4234 boxes and
  4556 lines; known differences: 33 at most 2/64 px (D14), 6 from shaping
  at a soft line end (D23), one of which moves a line break.
* `<img>` (M5b): PNG/JPEG from `data:` URLs or `Options.Assets`, aspect
  ratio sizing, integer bilinear scaling, rounded clipping. Layout matches
  Chromium; pixels differ by JPEG decoding/filtering (D29) and Chromium
  not clipping images to `border-radius` (D28).
* Default style sheet (`internal/render/ua.css`, D27) for links,
  buttons and inputs, with `:hover/:active/:focus/:disabled`; `disabled`
  buttons and inputs; `onfocus`/`onblur`; overlay scrollbars with
  draggable thumbs.
* Text inputs (M8a): `internal/edit` buffer (graphemes, words,
  selection), keyboard editing and movement, committed text events,
  clipboard copy/cut/paste (`App.SetClipboard`, headless in-memory),
  pointer caret placement and drag selection, horizontal scrolling,
  placeholder, `oninput`/`onchange`, controlled and uncontrolled values
  (`App.Value`), editing state kept across re-renders.
* Handlers and state (M5): `onclick="Name"` bound with `App.Handle`,
  bubbling with `Ctx.StopPropagation`, `Ctx.Attr`, click = press+release
  on the common ancestor, `SetViewFunc` re-run after handlers or
  `Invalidate`, missing-handler warnings. `headless.Session` steps
  synthetic events (Click, Press, Move, Wheel, Key, Resize) one frame at a
  time; `App.Bounds(id)` finds elements for tests.
* `view` (M5): keeps the document between frames and recomputes only
  changed stages (parse / style / layout / place / paint); style sheets
  and shaped text are cached across rebuilds; idle frames copy the
  retained image (about 11 µs at 280x220), a click with re-render about
  1.1 ms. Hit testing follows paint order exactly (shared traversal).
* Windows backend (`platform/windows`, x/sys/windows): Win32 window,
  per-monitor DPI awareness, frames drawn with `SetDIBitsToDevice`,
  mouse/wheel/keys/`WM_CHAR` text, clipboard, `windows.Run(app)`; seven
  staged commands with `EXPECT`/`RESULT` lines and `-debug` logs (shared
  content in `platform/internal/stagekit`); `examples/counter` and
  `examples/demo` take `-window` on Windows too. Verified by the user:
  stages 1-2 (Win32 bindings, window, presentation, 125% DPI sizing) on
  Windows build 26300; stages 1-7 pass and the demo runs.
* Demo app (M9, `examples/demo`): a contacts manager (searchable list
  with selection and scrolling, edit form with Save/Revert enabled only
  when dirty, delete confirmation modal with Escape and backdrop click,
  add with focus on the new name) written against the public API only.
  A scripted headless session produces 8 states compared byte for byte
  with goldens in `examples/demo/testdata` (also on linux/386);
  `go run ./demo -window` opens it on macOS (untested).
* Performance (demo, 214 contacts, 2560x1600 device pixels = 1280x800
  @2x, one event + one frame, 4-core cloud VM; `cd examples && go test
  ./demo -run xxx -bench Frame`):

  | case | before | after |
  |---|---|---|
  | idle | 1.6 ms | 0.08 ms |
  | hover change | 51 ms | 2.2 ms |
  | typing (re-render) | 56 ms | 14 ms |
  | scrolling 30px | 51 ms | 7.4 ms |

  From: border painting limited to the border ring, damage-region
  repainting and copying, incremental paint-only restyle, faster opaque
  fills (all byte-identical to full repaints; checked by
  `FLEXUI_CHECK_DAMAGE` in every app/demo test). Typing still re-parses,
  re-lays out and repaints the changed region (the HTML changes).
* Interaction (M6): hit testing in exact paint order (z-index, stacking
  contexts, overflow clips; checked pixel-by-pixel against the painted
  output on random documents), `:hover`/`:active`/`:focus` restyling only
  when the document uses that pseudo-class, focus by pointer and
  Tab/Shift+Tab (HTML tabindex order), Enter/Space activation,
  `onkeydown` with `PreventDefault`, `App.Focus`/`FocusedID`, scroll
  containers (`overflow: scroll`) with wheel (chaining) and drag
  scrolling, clamping, and focus and scroll offsets kept across
  re-renders by id or tree position (`App.ScrollOffset`).
* `render`: dom → cascade → layout → display list in CSS paint order
  (stacking contexts from `z-index` and `opacity`, negative/zero/positive
  layers, `overflow` clips carried to out-of-order contexts) → raster,
  on a white canvas.
* Tests:
  * 25 RenderHTML goldens (20 documents, 4 also at 2x) in
    `testdata/render/golden`. They include three z-index documents with
    overlapping absolute boxes (negative, nested and clipped contexts).
  * 8 rasterizer goldens.
  * The `div255` rounding is verified exhaustively.
* Pixel diff against Chromium (`TestChromeDiff`, `go test
  ./internal/render -run TestChromeDiff -v`). Errors are in levels of 255;
  ">8" and ">32" are the shares of pixels whose largest channel error
  exceeds that:

  | document | max | mean | >8 | >32 | verdict |
  |---|---|---|---|---|---|
  | app_shell | 30 | 0.078 | 0.096% | 0.000% | accepted (D17, D18) |
  | app_shell@2x | 56 | 0.103 | 0.069% | 0.016% | accepted (D17, D18) |
  | borders | 0 | 0.000 | 0% | 0% | exact |
  | boxes | 0 | 0.000 | 0% | 0% | exact |
  | cascade | 0 | 0.000 | 0% | 0% | exact |
  | fractional | 69 | 0.066 | 0.240% | 0.056% | accepted (D18) |
  | grid_wrap | 81 | 0.183 | 0.466% | 0.067% | accepted (D17, D18) |
  | opacity | 2 | 0.149 | 0% | 0% | accepted (D19) |
  | overflow | 57 | 0.120 | 0.631% | 0.148% | accepted (D18) |
  | radius | 63 | 0.170 | 0.950% | 0.128% | accepted (D18) |
  | radius@2x | 74 | 0.059 | 0.353% | 0.041% | accepted (D18) |
  | shadows | 85 | 0.473 | 1.184% | 0.055% | accepted (D17, D18) |
  | specificity | 34 | 0.053 | 0.456% | 0.003% | accepted (D18) |
  | text_card | 41 | 0.313 | 1.343% | 0.017% | accepted (D21, D18) |
  | text_card@2x | 203 | 0.599 | 1.530% | 0.696% | accepted (D22, D21) |
  | text_ellipsis | 48 | 0.394 | 1.682% | 0.105% | accepted (D21, D18) |
  | text_paragraphs | 47 | 0.505 | 2.179% | 0.081% | accepted (D21) |
  | controls | 67 | 0.160 | 0.831% | 0.044% | accepted (D18, D21) |
  | images | 184 | 0.334 | 0.232% | 0.199% | accepted (D28, D29) |
  | inline_text | 52 | 0.535 | 2.551% | 0.145% | accepted (D21) |
  | zindex_clip | 26 | 0.014 | 0.065% | 0% | ok |
  | zindex_clip@2x | 50 | 0.013 | 0.066% | 0.004% | accepted (D18) |
  | zindex_nested | 1 | 0.022 | 0% | 0% | ok |
  | zindex_overlap | 0 | 0.000 | 0% | 0% | exact |

  The remaining differences are anti-aliasing on curves (D18) and glyphs
  (D21), the box-blur shadow approximation (D17), ±2 opacity rounding
  (D19), and at 2x Chromium's device-pixel line metrics (D22).
  Side-by-side diff images confirmed that nothing structural differs.
* Binary size: `examples/hello` with both fonts is 7.2 MB (linux/amd64).
* Determinism: layout and rasterization are integer-only; tests pass with
  `GOARCH=386`.
* `scripts/check-portability.sh` (the portability gate): builds and vets
  the library, the macOS backend and its stage commands, and the
  examples module for 5 targets with `CGO_ENABLED=0`; checks the
  dependency rules on every target; and checks that the 24 golden
  renders hash identically on linux/386 (`testdata/render/hashes.txt`,
  `TestHashes`).
* macOS backend (`platform/darwin`, purego): window, layer
  presentation at the backing scale, resize, mouse/wheel/keys, typed
  text, system clipboard, `darwin.Run(app)`; seven staged commands with
  `EXPECT`/`RESULT` lines and `-debug` logs; `examples/counter -window`.
  Verified by the user on an Apple-silicon MacBook Air (macOS 27.2,
  scale 2): stages 1-7 pass, including a byte-identical presented frame
  (stage 3), live resizing, and typing into inputs with input/change
  events (stage 6). Not yet verified: the system clipboard and
  Option/Cmd caret keys, scale changes between displays, Intel Macs. Details in
  `platform/darwin/README.md`.

## Stubbed

* Text: Latin left-to-right only (no bidi or complex scripts), one
  bundled family, no italics (D30), no `letter-spacing`/`word-spacing`,
  no hyphenation, no `vertical-align`; inline elements cannot have
  padding, borders or margins (warning).
* `<img>`: no `object-fit`, no mipmaps (strong downscaling aliases), no
  animated images, no other formats.
* `<input>`: no IME composition, undo, double-click selection,
  autoscroll while selecting, `maxlength`/`readonly`/password; the caret
  does not blink (D31).
* Handlers: `onclick`, `onkeydown`, `onfocus`, `onblur`, `oninput`,
  `onchange`. Handlers must not call `App` methods that lock (`Focus`,
  `Value`, `Bounds`, ...): use `Ctx` (`Ctx.Focus` applies after the
  re-render).
* Scrolling: overlay scrollbars only (D25), no keyboard scrolling, no
  track clicks, no inertia; no scrolling of `body` itself unless it has
  `overflow: scroll`.

## Known limitations / bugs

* Layout deviations D11-D14 (`spec/SUBSET.md` §4): absolute boxes with
  both vertical insets that overflow, intrinsic sizes of wrapping
  containers with explicit `flex-basis`, some nested cyclic percentages,
  and 2/64 px rounding differences.
* Paint deviations D15-D20 (`spec/SUBSET.md` §6): one `border-color`, one
  outer `box-shadow`, box-blur shadows, supersampled curve AA, opacity
  rounding, and no body-to-canvas background propagation.
* Text deviations D21-D24 (`spec/SUBSET.md` §7.5): glyph AA differs from
  Skia's, layout is scale-independent (Chromium's 2x line heights differ),
  kerning at some soft line ends, and ellipsis on text directly inside an
  ellipsizing element.
* A changed view rebuilds the whole document (no tree diffing); text and
  style sheets are cached, but layout runs on the whole tree and the whole
  frame is repainted (no damage regions). `layout` allocates per
  container.
* Rasterizer speed: the shadow blur dominates (about half of a card
  frame).
* Glyph masks are cached per face (cleared at 32k entries).
* A change of the view's HTML (e.g. typing into a controlled input)
  re-parses, restyles and lays out the whole document (no tree diffing);
  about 14 ms for the demo at 2560x1600. Pseudo-class changes that affect
  layout (e.g. bold on hover) also rebuild everything.
