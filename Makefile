.PHONY: ci ci-unlocked fast-ci fast-ci-unlocked fast-checks fast-checks-unlocked slow-ci slow-ci-unlocked \
	generated generated-unlocked ast-boundary formatter-boundary formatter-ci formatter-ci-unlocked tgolint-boundary \
	lint lint-unlocked test test-unlocked unit-test unit-test-unlocked unit-test-fast unit-test-fast-unlocked \
	tgolint-unit-test tgolint-unit-test-unlocked e2e-test e2e-test-unlocked tgolint-test tgolint-test-unlocked \
	formatter-go-corpus formatter-go-corpus-unlocked adr tgofmt-check tgofmt-check-unlocked tgofmt-check-test \
	allocation-test allocation-test-unlocked dogfood dogfood-unlocked markdown source-size tgo-placeholders \
	upstream-provenance pre-commit-boundary vscode-test vscode-test-unlocked build build-unlocked install-hooks install-tools

ci:
	+@scripts/with-local-validation-lock.sh $(MAKE) -j2 ci-unlocked

ci-unlocked: fast-ci-unlocked

fast-ci:
	+@scripts/with-local-validation-lock.sh $(MAKE) -j2 fast-ci-unlocked

fast-ci-unlocked: fast-checks-unlocked unit-test-fast-unlocked e2e-test-unlocked allocation-test-unlocked

fast-checks:
	+@scripts/with-local-validation-lock.sh $(MAKE) fast-checks-unlocked

fast-checks-unlocked: generated-unlocked ast-boundary formatter-boundary formatter-ci-unlocked tgolint-boundary dogfood-unlocked lint-unlocked markdown adr source-size tgo-placeholders upstream-provenance pre-commit-boundary tgofmt-check-unlocked tgofmt-check-test

slow-ci:
	+@scripts/with-local-validation-lock.sh $(MAKE) -j2 slow-ci-unlocked

slow-ci-unlocked: tgolint-unit-test-unlocked formatter-go-corpus-unlocked

ast-boundary:
	python3 scripts/check_ast_boundary.py

formatter-boundary:
	python3 scripts/check_formatter_boundary.py

formatter-ci:
	+@scripts/with-local-validation-lock.sh $(MAKE) formatter-ci-unlocked

formatter-ci-unlocked:
	python3 -m unittest scripts.formatter_ci_test scripts.local_validation_lock_test
	python3 scripts/formatter_ci.py verify

formatter-go-corpus:
	+@scripts/with-local-validation-lock.sh $(MAKE) formatter-go-corpus-unlocked

formatter-go-corpus-unlocked:
	TGO_FULL_GO_FORMAT_CORPUS=1 go test ./pkg/format -run TestSourceMatchesFullGoTree -count=1 && \
		{ test -z "$$GITHUB_OUTPUT" || echo "passed=true" >> "$$GITHUB_OUTPUT"; }

tgofmt-check:
	+@scripts/with-local-validation-lock.sh $(MAKE) tgofmt-check-unlocked

tgofmt-check-unlocked:
	python3 -m scripts.check_tgofmt

tgofmt-check-test:
	python3 -m unittest scripts.check_tgofmt_test

tgolint-boundary:
	python3 scripts/check_tgolint_boundary.py

dogfood:
	+@scripts/with-local-validation-lock.sh $(MAKE) dogfood-unlocked

dogfood-unlocked:
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

tgo-placeholders:
	python3 -m unittest scripts.check_tgo_placeholders_test
	python3 scripts/check_tgo_placeholders.py

upstream-provenance:
	python3 -m unittest scripts.check_upstream_provenance_test
	python3 scripts/check_upstream_provenance.py

pre-commit-boundary:
	python3 scripts/check_pre_commit.py

vscode-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) vscode-test-unlocked

vscode-test-unlocked:
	./vscode.sh --package-only
	python3 editors/vscode/test/package.py editors/vscode/tgo-navigation.vsix
	cd editors/vscode && xvfb-run -a npm run test:all

generated:
	+@scripts/with-local-validation-lock.sh $(MAKE) generated-unlocked

generated-unlocked:
	python3 -m unittest scripts.check_generated_test
	python3 scripts/check_generated.py

lint:
	+@scripts/with-local-validation-lock.sh $(MAKE) lint-unlocked

lint-unlocked:
	golangci-lint run ./...
	golangci-lint fmt --diff

test:
	+@scripts/with-local-validation-lock.sh $(MAKE) test-unlocked

test-unlocked: unit-test-unlocked e2e-test-unlocked tgolint-test-unlocked allocation-test-unlocked

unit-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) unit-test-unlocked

unit-test-unlocked:
	go test ./...

unit-test-fast:
	+@scripts/with-local-validation-lock.sh $(MAKE) unit-test-fast-unlocked

unit-test-fast-unlocked:
	go test $$(go list ./... | grep -vx 'tgo/internal/tgolint')

tgolint-unit-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) tgolint-unit-test-unlocked

tgolint-unit-test-unlocked:
	go test ./internal/tgolint

e2e-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) e2e-test-unlocked

e2e-test-unlocked:
	python3 tests/e2e/run.py

tgolint-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) tgolint-test-unlocked

tgolint-test-unlocked:
	python3 tests/tgolint/run.py

allocation-test:
	+@scripts/with-local-validation-lock.sh $(MAKE) allocation-test-unlocked

allocation-test-unlocked:
	python3 tests/allocations/run.py

build:
	+@scripts/with-local-validation-lock.sh $(MAKE) build-unlocked

build-unlocked:
	go build -o bin/tgo ./cmd/tgo
	go build -o bin/tgofmt ./cmd/tgofmt
	go build -o bin/tgolint ./cmd/tgolint
	go build -o bin/tgonav ./cmd/tgonav

install-tools:
	sh scripts/install-lint.sh

install-hooks:
	git config --local core.hooksPath .githooks
