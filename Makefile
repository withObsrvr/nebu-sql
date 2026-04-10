GO ?= go

.PHONY: test build run fmt smoke-real

test:
	$(GO) test ./...

build:
	$(GO) build ./...

run:
	$(GO) run ./cmd/nebu-sql

fmt:
	gofmt -w ./cmd ./internal

smoke-real:
	./scripts/smoke-real-processors.sh
