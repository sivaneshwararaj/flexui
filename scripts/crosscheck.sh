#!/usr/bin/env bash
# Regenerates the Chromium oracle data with headless Chromium (needs node
# with the playwright package):
#  * testdata/layout/chrome.json: element rectangles of the layout cases;
#  * testdata/render/chrome: screenshots of the RenderHTML documents;
#  * testdata/text: text layout (boxes and lines) of the text documents.
# Pages are set up by scripts/flexui-page.cjs (base sheet, bundled fonts).
set -euo pipefail
cd "$(dirname "$0")/.."
tmp="$(mktemp -d)"
trap 'rm -r "$tmp"' EXIT
go test ./internal/layout -run '^TestEmitHTML$' -count=1 -args -emit-html="$tmp/html" >/dev/null
node scripts/crosscheck.cjs "$tmp/html" >"$tmp/chrome.json"
mv "$tmp/chrome.json" testdata/layout/chrome.json
echo "wrote testdata/layout/chrome.json ($(grep -c '^  "' testdata/layout/chrome.json) cases)"

# RenderHTML documents: Chromium screenshots for TestChromeDiff.
hidpi=$(grep -o 'hidpiDocs = map\[string\]bool{[^}]*}' internal/render/golden_test.go | grep -o '"[a-z_0-9]*"' | tr -d '"' | paste -sd, -)
node scripts/screenshot.cjs testdata/render/docs testdata/render/chrome "$hidpi"
echo "wrote testdata/render/chrome ($(ls testdata/render/chrome | wc -l) screenshots)"

# Text layout: hand-written documents and the seeded random ones.
node scripts/textcheck.cjs testdata/text/docs >testdata/text/chrome.json
go test ./internal/render -run '^TestEmitText$' -count=1 -args -emit-text="$tmp/text" -emit-inline="$tmp/inline" -emit-script="$tmp/script" -emit-bidi="$tmp/bidi" >/dev/null
node scripts/textcheck.cjs "$tmp/text" >testdata/text/random.json
node scripts/textcheck.cjs "$tmp/inline" >testdata/text/inline.json
node scripts/textcheck.cjs "$tmp/script" >testdata/text/script.json
node scripts/textcheck.cjs "$tmp/bidi" >testdata/text/bidi.json
echo "wrote testdata/text/chrome.json, random.json, inline.json, script.json and bidi.json"
