.PHONY: ci lint test build install-hooks install-tools

ci: lint test
	python3 scripts/check_markdown.py

lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

test:
	go test ./...
	python3 tests/e2e/run.py
	python3 tests/tgolint/run.py

build:
	go build -o bin/tgo ./cmd/tgo
	go build -o bin/tgolint ./cmd/tgolint

install-tools:
	sh scripts/install-lint.sh

install-hooks:
	git config --local core.hooksPath .githooks
