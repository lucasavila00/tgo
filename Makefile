GOMARKDOC = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0
DOC_PACKAGES = ./cmd/tgo ./pkg/transpiler ./internal/driver ./internal/checker ./internal/frontend ./internal/printer ./internal/format ./internal/frontend/parser ./internal/frontend/scanner ./internal/lowering ./internal/lowering/shared ./internal/lowering/propagation ./internal/lowering/comprehensions ./internal/lowering/valueexpr ./internal/lowering/source ./internal/lowering/names ./internal/controlflow ./internal/lowering/enums ./internal/sourcefile

.PHONY: prepare docs docs-check lint

prepare:
	sh scripts/prepare-ast.sh
	sh scripts/prepare-checker.sh
	sh scripts/prepare-frontend.sh

docs: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) "$$package"; done

docs-check: prepare
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) --check "$$package"; done

lint: prepare
	go run ./scripts/check-source-size.go
	go run ./scripts/check-function-comments
	sh scripts/check-dead-code.sh
	golangci-lint run ./...
	golangci-lint fmt --diff ./...
