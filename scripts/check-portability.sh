#!/usr/bin/env bash
# Builds and vets every package for every supported target with cgo
# disabled. Must pass from a Linux machine.
set -euo pipefail
cd "$(dirname "$0")/.."

targets="linux/amd64 linux/arm64 darwin/arm64 darwin/amd64 windows/amd64"
status=0
for t in $targets; do
	if GOOS="${t%/*}" GOARCH="${t#*/}" CGO_ENABLED=0 go build ./... &&
		GOOS="${t%/*}" GOARCH="${t#*/}" CGO_ENABLED=0 go vet ./...; then
		echo "ok   $t"
	else
		echo "FAIL $t"
		status=1
	fi
done

# Core packages must not depend on platform code or OS-specific packages.
for p in geom style layout dom paint text app; do
	if GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go list -deps "./$p" |
		grep -E 'flexui/platform|golang.org/x/sys|ebitengine/purego' >/dev/null; then
		echo "FAIL core package $p imports platform code"
		status=1
	fi
done
exit $status
