# M5b review images

* `contact_sheet.png` (from `m5b_test.go`, synthetic events): the base
  page (inline badge, wrapped inline link, image from `Options.Assets`
  with rounded corners, default input/button styles, disabled button,
  overlay scrollbar), link hover (both fragments), Save focused after a
  click, and the log scrolled by dragging its scrollbar thumb. The single
  frames are `m5b_*.png`.
* `*_vs_chromium.png`: flexui | Chromium | difference, at 2x zoom, for
  the RenderHTML documents `inline_text` (layout identical; glyph AA),
  `images` (layout identical; JPEG decoding, scaling filter, and
  Chromium not clipping to border-radius) and `controls` (default
  styles; curve and glyph AA).
