#!/usr/bin/env bash
# The portability gate (M7). Builds and vets every package (library,
# macOS backend and its stage commands, examples module) for every
# supported target with cgo disabled, checks the dependency rules, and
# checks that rendering is byte-identical on another architecture.
# Must pass from a Linux machine.
set -euo pipefail
cd "$(dirname "$0")/.."

targets="linux/amd64 linux/arm64 darwin/arm64 darwin/amd64 windows/amd64 windows/arm64"
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
# Checked for every OS, since files can be OS-specific.
for t in $targets; do
	for p in . ./internal/...; do
		if GOOS="${t%/*}" GOARCH="${t#*/}" CGO_ENABLED=0 go list -deps "$p" |
			grep -E 'flexui/platform/|golang.org/x/sys|ebitengine/purego' >/dev/null; then
			echo "FAIL $p imports a platform backend or OS-specific package ($t)"
			status=1
		fi
	done
done
# purego only in the macOS backend and x/sys only in the Windows backend:
# the headless platform, the platform interfaces and the other backend
# must not pull them in on any OS.
for t in $targets; do
	for p in ./platform ./platform/headless ./platform/windows; do
		if GOOS="${t%/*}" GOARCH="${t#*/}" CGO_ENABLED=0 go list -deps "$p" | grep -q ebitengine/purego; then
			echo "FAIL $p depends on purego ($t)"
			status=1
		fi
	done
	for p in ./platform ./platform/headless ./platform/darwin; do
		if GOOS="${t%/*}" GOARCH="${t#*/}" CGO_ENABLED=0 go list -deps "$p" | grep -q golang.org/x/sys; then
			echo "FAIL $p depends on golang.org/x/sys ($t)"
			status=1
		fi
	done
done
echo "ok   dependency rules"

# Determinism: the golden documents render to the recorded hashes on a
# different architecture (32-bit) too. (On macOS/Windows, run TestHashes
# there.)
if GOARCH=386 CGO_ENABLED=0 go test -count=1 ./internal/render -run '^TestHashes$' >/dev/null; then
	echo "ok   render hashes on linux/386"
else
	echo "FAIL render hashes differ on linux/386"
	status=1
fi
exit $status
