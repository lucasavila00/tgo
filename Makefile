GOMARKDOC = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0
DOC_PACKAGES = ./cmd/tgo ./pkg/transpiler ./internal/driver ./internal/frontend ./internal/lowering

.PHONY: docs docs-check lint

docs:
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) "$$package"; done

docs-check:
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) --check "$$package"; done

lint:
	go run ./scripts/check-source-size.go
	golangci-lint run ./...
	golangci-lint fmt --diff ./...
