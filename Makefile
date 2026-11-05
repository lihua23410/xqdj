GO ?= go
.PHONY: audit build test run
audit:
	$(GO) run ./cmd/audit
build: audit
	$(GO) build -o xqdj .
test: audit
	$(GO) test ./...
run: audit
	$(GO) run .
