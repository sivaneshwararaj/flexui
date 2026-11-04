#!/usr/bin/env bash
# Regenerates testdata/layout/chrome.json: renders every layout test case
# in headless Chromium and records the element rectangles. Requires node
# with the playwright package and a Chromium it can launch.
set -euo pipefail
cd "$(dirname "$0")/.."
tmp="$(mktemp -d)"
trap 'rm -r "$tmp"' EXIT
go test ./layout -run '^TestEmitHTML$' -count=1 -args -emit-html="$tmp/html" >/dev/null
node scripts/crosscheck.cjs "$tmp/html" >"$tmp/chrome.json"
mv "$tmp/chrome.json" testdata/layout/chrome.json
echo "wrote testdata/layout/chrome.json ($(grep -c '^  "' testdata/layout/chrome.json) cases)"
