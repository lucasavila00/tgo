GOMARKDOC = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0
DOC_PACKAGES = ./cmd/tgo ./pkg/transpiler ./internal/driver ./internal/frontend ./internal/lowering

.PHONY: prepare build docs docs-check

prepare:
	@sh scripts/prepare-checker.sh

build: prepare
	go build ./...

docs: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) "$$package"; done

docs-check: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) --check "$$package"; done
