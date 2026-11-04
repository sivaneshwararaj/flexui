.PHONY: all check test vet fmt portability crosscheck

all: check

# Pre-commit checklist.
check: fmt vet test portability

test:
	go test ./...
	cd examples && go vet ./... && go test ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

portability:
	./scripts/check-portability.sh

# Regenerate the Chromium oracle data (layout, screenshots, text) from
# headless Chromium (needs node + playwright). Only needed when test
# documents or the oracle setup change.
crosscheck:
	./scripts/crosscheck.sh
