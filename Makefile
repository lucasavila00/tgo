.PHONY: ci ci-unlocked generated lint test unit-test e2e-test tgolint-test \
	markdown build install-hooks install-tools

ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 ci-unlocked

ci-unlocked: generated e2e-test tgolint-test lint unit-test markdown

markdown:
	python3 scripts/check_markdown.py

generated:
	python3 scripts/check_generated.py

lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

test: unit-test e2e-test tgolint-test

unit-test:
	go test ./...

e2e-test:
	python3 tests/e2e/run.py

tgolint-test:
	python3 tests/tgolint/run.py

build:
	go build -o bin/tgo ./cmd/tgo
	go build -o bin/tgolint ./cmd/tgolint

install-tools:
	sh scripts/install-lint.sh

install-hooks:
	git config --local core.hooksPath .githooks
