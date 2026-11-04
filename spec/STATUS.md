# Status

| Milestone | State |
|---|---|
| M0 Spec and scaffolding | done |
| M1 Layout engine | in progress |
| M2 Rasterizer | not started |
| M3 HTML+CSS to pixels | not started |
| M4 Text | not started |
| M5 Handlers and state | not started |
| M6 Interaction | not started |
| M7 Portability gate + macOS | portability script exists; darwin not started |
| M8 Text input | not started |
| M9 Demo app | not started |

## Works

* `style`: declaration parsing for all layout properties of the subset,
  shorthands (`margin`, `padding`, `border-width`, `gap`, `flex`), px/%/rem,
  warnings for unsupported input.
* `platform`: interface; `headless` platform delivers events and renders
  frames into an image.
* `scripts/check-portability.sh`: builds and vets 5 targets with
  `CGO_ENABLED=0` and checks the core import rule.

## Stubbed

* `dom`, `paint`, `text`, `app`: package docs only.

## Known bugs

None recorded.
