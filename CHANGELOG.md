# Changelog

All notable changes are recorded here. The project uses semantic
versioning with `v0.x` tags until the API settles.

## Unreleased

### Added
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
