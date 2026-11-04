# M4 review images

* `text_card.png`, `text_card@2x.png`, `text_ellipsis.png`,
  `text_paragraphs.png`: RenderHTML output (the goldens).
* `*_vs_chromium.png`: flexui | Chromium | difference (red, x4), at 2x
  zoom. Glyph positions agree; differences are anti-aliasing inside glyph
  edges (D21) and curve AA on rounded corners (D18).
