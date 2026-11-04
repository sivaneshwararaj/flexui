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

### Changed
- Implementation packages moved under `internal/`.
