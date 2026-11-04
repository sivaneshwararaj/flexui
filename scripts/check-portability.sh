#!/usr/bin/env bash
# Builds and vets every package (library and examples module) for every
# supported target with cgo disabled. Must pass from a Linux machine.
set -euo pipefail
cd "$(dirname "$0")/.."

targets="linux/amd64 linux/arm64 darwin/arm64 darwin/amd64 windows/amd64"
status=0
out="$(mktemp -d)"
trap 'rm -r "$out"' EXIT
check() { # dir goos goarch [build flags...]
	local dir="$1" os="$2" arch="$3"
	shift 3
	(cd "$dir" && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build "$@" ./... &&
		GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go vet ./...)
}
for t in $targets; do
	if check . "${t%/*}" "${t#*/}" && check examples "${t%/*}" "${t#*/}" -o "$out"; then
		echo "ok   $t"
	else
		echo "FAIL $t"
		status=1
	fi
done

# The root package and internal/ must not depend on platform backends or
# OS-specific packages (the platform interface package itself is allowed).
for p in . ./internal/...; do
	if GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go list -deps "$p" |
		grep -E 'flexui/platform/|golang.org/x/sys|ebitengine/purego' >/dev/null; then
		echo "FAIL $p imports a platform backend or OS-specific package"
		status=1
	fi
done
exit $status
