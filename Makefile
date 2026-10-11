GOMARKDOC = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0
DOC_PACKAGES = ./cmd/tgo ./pkg/transpiler ./internal/driver ./internal/checker ./internal/frontend ./internal/lowering ./internal/controlflow

.PHONY: prepare build docs docs-check lint

prepare:
	@sh scripts/prepare-checker.sh

build: prepare
	go build ./...

docs: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) "$$package"; done

docs-check: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) --check "$$package"; done

lint: prepare
	go run ./scripts/check-source-size.go
	golangci-lint run ./...
	golangci-lint fmt --diff ./...
