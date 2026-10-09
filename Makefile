.PHONY: ci ci-unlocked fast-ci fast-ci-unlocked slow-ci slow-ci-unlocked \
	generated ast-boundary lint test unit-test e2e-test tgolint-test \
	allocation-test dogfood markdown tgo-size build install-hooks install-tools

ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 ci-unlocked

ci-unlocked: fast-ci-unlocked slow-ci-unlocked

fast-ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 fast-ci-unlocked

fast-ci-unlocked: generated ast-boundary dogfood lint markdown tgo-size

slow-ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 slow-ci-unlocked

slow-ci-unlocked: unit-test e2e-test tgolint-test allocation-test

ast-boundary:
	python3 scripts/check_ast_boundary.py

dogfood:
	! rg -n '//[[:space:]]*tgolint:ignore' cmd internal pkg
	go run ./cmd/tgolint ./cmd/... ./internal/... ./pkg/...

markdown:
	python3 scripts/check_markdown.py

tgo-size:
	python3 scripts/check_tgo_size.py

generated:
	python3 scripts/check_generated.py

lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

test: unit-test e2e-test tgolint-test allocation-test

unit-test:
	go test ./...

e2e-test:
	python3 tests/e2e/run.py

tgolint-test:
	python3 tests/tgolint/run.py

allocation-test:
	python3 tests/allocations/run.py

build:
	go build -o bin/tgo ./cmd/tgo
	go build -o bin/tgofmt ./cmd/tgofmt
	go build -o bin/tgolint ./cmd/tgolint

install-tools:
	sh scripts/install-lint.sh

install-hooks:
	git config --local core.hooksPath .githooks
