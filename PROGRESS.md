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
| 6. Bidi (UAX #9), `dir`/`direction`, RTL alignment, Arabic/Hebrew packs | **in progress**: text and layout done and checked; docs, random docs, edge cases left (see below) |
| 7. Cluster-aware caret/selection/backspace, bidi visual movement | not started |
| 8. IME composition events + caret rect (headless) | not started |
| 9. IME on macOS (NSTextInputClient) and Windows (IMM32), staged | not started |
| 10. CJK font pack (opt-in) + UAX #14 for ideographs | not started |
| `testdata/multilingual` (Tamil, Hindi, mixed done; Arabic, Hebrew, mixed RTL, long paragraphs to add) | partly |
| Final summary to the user (changed / unverified / weakest) | at the end |

Waiting on the user: stage 8 (multilingual clipboard) on macOS and
Windows: `go run ./platform/<darwin|windows>/stages/stage8`.

## Step 6 notes (current)

Done so far (checkpoint commit "Bidi part 1"):

* `fonts/arabic` (Noto Sans Arabic 2.013), `fonts/hebrew` (Noto Sans
  Hebrew 3.001), in the Chromium harness and the test imports.
* Style: `direction: ltr|rtl` (inherited), the `dir` attribute
  (`ltr`/`rtl`, a presentational hint), `text-align: start|end` (start
  is the initial value; resolved with `TextAlign.Physical`).
* `internal/text/bidi.go`: levels per hard line with go-text's bidi
  (per-character levels read from an unexported field, `runLevels`,
  guarded by `TestRunLevels`: go-text's `Runs` merge levels 0 and 2);
  items and pieces split at level changes; RTL items shaped RTL (glyphs
  stored in logical order); lines reordered (L1, L2) into visual
  segments `Line.vis`; `X`, `Spans` (inline box fragments) and
  `Glyphs` use them. All-LTR text keeps the old code paths.
* Layout: flex axes in RTL containers (`mainRev`, `crossRev` in
  `internal/layout/flex.go`), over-constrained absolute boxes and
  relative offsets (right wins in RTL), the space-around/evenly overflow
  fallback. 400 random `rtl_` layout cases against Chromium: all match
  except 6 that also differ without direction (D11, D12, D14).
* Hand-written `testdata/text/docs/bidi_*.html` match Chromium exactly.

Still to do for step 6:

* Seeded random bidi text documents vs Chromium (like `script_`).
* Check against Chromium: trailing-space reshaping (`reshapeEnd`) in RTL
  lines, ellipsis in RTL lines, `dir` on inline elements (browsers make
  it `unicode-bidi: isolate`; currently ignored for inline elements).
* `<input>` with `dir="rtl"` (alignment, caret) — or leave to step 7.
* `testdata/multilingual`: Arabic, Hebrew, mixed RTL pages; review PNGs
  in `review/step6`; SUBSET.md §7.6 (bidi) and deviations; STATUS,
  CHANGELOG, GUIDE.

## How the Chromium comparison works

`make crosscheck` (needs node + playwright + chromium) regenerates
`testdata/layout/chrome.json`, `testdata/render/chrome/`,
`testdata/text/*.json`. To add hand-written text docs: put them in
`testdata/text/docs/` and run
`node scripts/textcheck.cjs testdata/text/docs > testdata/text/chrome.json`.
`go test ./internal/render -run TestTextChrome` compares.
