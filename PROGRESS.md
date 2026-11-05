# Progress and resume point

This file is updated with every push so work can resume from any
machine (local Claude Code or a cloud session). Read it first, then
`AGENTS.md`, `claude.md` and `spec/STATUS.md`.

## Rules (from the user)

* Commit after each step; before each commit run `make check` (gofmt,
  vet, tests in both modules, `scripts/check-portability.sh`) and update
  `spec/STATUS.md`.
* Never regenerate a golden or loosen a tolerance just to pass. Differences
  from Chromium: fix, or record a numbered deviation in `spec/SUBSET.md`.
* If a gate fails after several honest attempts, stop and report.
* Representative PNGs go in `review/<step>`.
* Anything needing a real Mac/Windows: staged command with EXPECT/RESULT
  lines and `-debug`, plus a README list of what is untested. Never claim
  it works without the user's report.
* Verify library behavior (go-text, bidi, purego, Windows APIs) with real
  output, not memory.
* Branch: `claude/brave-hawking-8qq1ua`. No PR unless asked.

## Task list: multilingual text

| Step | State |
|---|---|
| 0. Tag `v0.1.0`, CHANGELOG | done (tag exists at ab4d228; push it from a local clone: `git tag -a v0.1.0 ab4d228 -m v0.1.0 && git push origin v0.1.0`, the cloud session cannot push tags) |
| 1-4. Tamil paste diagnosis | done: clipboard and buffer correct; word movement in Indic text fixed; drawing was the fault (no glyphs) |
| 5. Itemization, font fallback, Noto Tamil/Devanagari packs | done (commit 43db1cd), matches Chromium |
| 6. Bidi (UAX #9), `dir`/`direction`, RTL alignment, Arabic/Hebrew packs | done (spec/SUBSET.md §7.6; D33 lists the residuals) |
| 7. Cluster-aware caret/selection/backspace, bidi movement | done (logical movement as Chromium; see spec/STATUS.md) |
| 8. IME composition events + caret rect (headless) | **next** |
| 9. IME on macOS (NSTextInputClient) and Windows (IMM32), staged | not started |
| 10. CJK font pack (opt-in) + UAX #14 for ideographs | not started |
| `testdata/multilingual` (Tamil, Hindi, mixed, Arabic, Hebrew, bidi mixed done; long paragraphs and CJK to add) | partly |
| Final summary to the user (changed / unverified / weakest) | at the end |

Waiting on the user: stage 8 (multilingual clipboard) on macOS and
Windows: `go run ./platform/<darwin|windows>/stages/stage8`.

## Step 8 notes (next)

* Add to `platform.Event`: composition kinds (start, update with the
  marked text, marked range/selection and underline style, commit,
  cancel). Add a caret-rectangle query the platform can call (the
  focused input's caret rect in window coordinates) so the OS candidate
  window can be placed.
* `internal/view`: an input in composition shows the marked text at the
  caret, underlined, not yet in the value (no `input` event until
  commit; Chromium does fire `input` during composition with
  `isComposing` — verify in Chromium before choosing).
* Headless: `headless.Session` methods to send composition events;
  goldens of the marked text.

## Step 7 summary (done)

* `internal/edit`: Backspace by code point except emoji clusters
  (`emojiCluster`), checked against Chromium (`TestChromiumEditing`).
* `internal/render/input.go`: inputs get the element's direction; RTL
  right-aligned (`inputOffset`), `InputScrollRange`, `InputRTL`,
  selections with `Text.Spans`. `internal/view/input.go`: arrows
  reversed in RTL inputs; scroll range both sides.
* `Text.visX`: logical line ends drawn on the start/end side.

## Step 6 summary (done)

* `internal/text/bidi.go` (levels, isolates as LRI/RLI/PDI, reordering
  into `Line.vis`, `Spans`, `Ink`), `dom.Node.Dir` (dir=auto via HTML
  directionality), `style.Direction`/`Isolate`, layout `mainRev` /
  `crossRev`. Random bidi documents: `TestTextChromeBidi`
  (`testdata/text/bidi.json`, per-grapheme positions); random RTL
  layout: `rtl_` cases in `internal/layout`.

## How the Chromium comparison works

`make crosscheck` (needs node + playwright + chromium) regenerates
`testdata/layout/chrome.json`, `testdata/render/chrome/`,
`testdata/text/*.json`. To add hand-written text docs: put them in
`testdata/text/docs/` and run
`node scripts/textcheck.cjs testdata/text/docs > testdata/text/chrome.json`.
`go test ./internal/render -run TestTextChrome` compares.
