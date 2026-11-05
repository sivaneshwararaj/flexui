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
| 8. IME composition events + caret rect (headless) | done (spec/SUBSET.md §8.7) |
| 9. IME on macOS (NSTextInputClient) and Windows (IMM32), staged | done, compile-checked only; user to run stage 9 on both OSes (and stage 6 again on macOS) |
| 10. CJK font pack (opt-in) + UAX #14 for ideographs | done (fonts/cjk; D34 for punctuation trimming) |
| `testdata/multilingual` (Tamil, Hindi, mixed, Arabic, Hebrew, bidi, CJK, long paragraphs, inputs, IME) | done |
| Final summary to the user (changed / unverified / weakest) | done (in the session; also spec/STATUS.md) |

macOS results (user): stage 6 with font packs and stage 8 pass; stage 9
partial (no `compose` RESULT lines yet: type with the Japanese input
method, not paste). `stage9 -auto` checks the IME methods without a
keyboard (sends the input-method messages itself). Windows stages 8 and 9 not run.

Waiting on the user (run on each OS, send the RESULT lines):
`go run ./platform/<darwin|windows>/stages/stage8` (multilingual
clipboard), `.../stage9` (IME, with a Japanese input method). macOS stage 6 re-run:
typing works (font packs were missing in the stage; fixed).

## Possible next work (not requested yet)

* D34: Blink's HanKerning (text-spacing-trim) with the `halt` feature.
* Korean pack (Noto Sans KR subset), CJK bold, Japanese forms (JP subset).
* `RenderHTML` does not draw `<input>` values (only apps do).
* Results of stages 8 and 9 (and stage 6 on macOS) from the user.

## Step 10 summary (done)

* `fonts/cjk` (Regular only; `Pack.Bold` may be nil), harness support
  for single-face packs, `script_cjk` docs, line-fit rule (`fitSpan`).

## Step 9 summary (done, untested)

* Windows `platform/windows/ime_windows.go` (IMM32); macOS
  `platform/darwin/ime_darwin.go` + `ime_darwin_{arm64,amd64}.s`;
  stage 9 on both (stagekit/ime.go: App9, LogIME). READMEs list risks.

## Step 8 summary (done)

* platform.EventCompose/Commit/Cancel, CaretRecter; view.Compose/
  Commit/CancelComposition/CaretRect; render underline (MarkStart/End);
  App.CaretRect; Ctx.Composing; headless Session methods;
  TestComposition replays the Chromium-measured sequence.

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
