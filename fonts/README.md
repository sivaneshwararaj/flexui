# Optional font packs

flexui draws text with its built-in Inter (Latin, Greek, Cyrillic).
Text in other scripts needs a fallback font: import the pack for each
script your app shows. Each pack embeds a Regular and a Bold face and
registers them in a fixed fallback order (independent of import order):

| Package | Font | Version | Scripts | Added binary size |
|---|---|---|---|---|
| `fonts/tamil` | Noto Sans Tamil | 2.004 | Tamil | +95 KB |
| `fonts/devanagari` | Noto Sans Devanagari | 2.007 | Devanagari (Hindi, Marathi, Nepali, ...) | +364 KB |
| `fonts/arabic` | Noto Sans Arabic | 2.013 | Arabic (Arabic, Persian, Urdu, ...), right to left | +288 KB |
| `fonts/hebrew` | Noto Sans Hebrew | 3.001 | Hebrew, right to left | +38 KB |

Binary sizes were measured on linux/amd64 with Go 1.24 for a program
calling `flexui.RenderHTML`: 7,442,947 bytes without packs, 7,538,191
with Tamil, 7,902,565 with Tamil and Devanagari; after the bidi work
(step 6) 7,563,563 without packs, 7,851,533 with Arabic, 7,601,669 with
Hebrew. Fonts are parsed on first use, not at startup.

```go
import (
	_ "github.com/sivaneshwararaj/flexui/fonts/devanagari"
	_ "github.com/sivaneshwararaj/flexui/fonts/tamil"
)
```

Font selection follows Chromium: each item (a run of one style and one
script) is shaped with Inter; clusters Inter has no glyph for are shaped
again with the next registered font, and so on (spec/SUBSET.md §7.3).
Text no registered font covers is drawn as .notdef boxes.

The fonts are the unhinted TrueType files from
https://github.com/notofonts/notofonts.github.io (`fonts/<Family>/unhinted/ttf/`),
unmodified, licensed under the SIL Open Font License 1.1 (`OFL.txt` in
each directory; it applies to the font files only).
