.PHONY: all check test vet fmt portability crosscheck

all: check

# Pre-commit checklist.
check: fmt vet test portability

test:
	go test ./...
	cd examples && go vet ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

portability:
	./scripts/check-portability.sh

# Regenerate testdata/layout/chrome.json from headless Chromium (needs node +
# playwright). Only needed when layout test cases are added or changed.
crosscheck:
	./scripts/crosscheck.sh
