.PHONY: ci ci-unlocked fast-ci fast-ci-unlocked fast-checks slow-ci slow-ci-unlocked \
	generated ast-boundary formatter-boundary tgolint-boundary lint test unit-test unit-test-fast tgolint-unit-test \
	e2e-test tgolint-test formatter-go-corpus adr tgofmt-check tgofmt-check-test \
	allocation-test dogfood markdown source-size pre-commit-boundary vscode-test build install-hooks install-tools

ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 ci-unlocked

ci-unlocked: fast-ci-unlocked

fast-ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 fast-ci-unlocked

fast-ci-unlocked: fast-checks unit-test-fast e2e-test allocation-test

fast-checks: generated ast-boundary formatter-boundary tgolint-boundary dogfood lint markdown adr source-size pre-commit-boundary tgofmt-check tgofmt-check-test

slow-ci:
	flock "$$(git rev-parse --git-path tgo-ci.lock)" $(MAKE) -j2 slow-ci-unlocked

slow-ci-unlocked: tgolint-unit-test formatter-go-corpus

ast-boundary:
	python3 scripts/check_ast_boundary.py

formatter-boundary:
	python3 scripts/check_formatter_boundary.py

formatter-go-corpus:
	TGO_FULL_GO_FORMAT_CORPUS=1 go test ./pkg/format -run TestSourceMatchesFullGoTree -count=1

tgofmt-check:
	python3 -m scripts.check_tgofmt

tgofmt-check-test:
	python3 -m unittest scripts.check_tgofmt_test

tgolint-boundary:
	python3 scripts/check_tgolint_boundary.py

dogfood:
	! grep -R -n -E --include='*.go' --include='*.tgo' \
		'//[[:space:]]*tgolint:ignore' cmd internal pkg
	go run ./cmd/tgolint ./cmd/... ./internal/... ./pkg/...

markdown:
	python3 scripts/check_markdown.py

adr:
	python3 -m unittest scripts.check_adrs_test
	python3 scripts/check_adrs.py

source-size:
	python3 -m unittest scripts.check_source_size_test
	python3 scripts/check_source_size.py

pre-commit-boundary:
	python3 scripts/check_pre_commit.py

vscode-test:
	./vscode.sh --package-only
	python3 editors/vscode/test/package.py editors/vscode/tgo-navigation.vsix
	cd editors/vscode && xvfb-run -a npm run test:all

generated:
	python3 -m unittest scripts.check_generated_test
	python3 scripts/check_generated.py

lint:
	golangci-lint run ./...
	golangci-lint fmt --diff

test: unit-test e2e-test tgolint-test allocation-test

unit-test:
	go test ./...

unit-test-fast:
	go test $$(go list ./... | grep -vx 'tgo/internal/tgolint')

tgolint-unit-test:
	go test ./internal/tgolint

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
	go build -o bin/tgonav ./cmd/tgonav

install-tools:
	sh scripts/install-lint.sh

install-hooks:
	git config --local core.hooksPath .githooks
