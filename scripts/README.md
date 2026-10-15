# Compiler preparation and source checks

`prepare-checker.sh` builds private Go checker source from a pinned version and
patch. `prepare-frontend.sh` builds the private parser and scanner with the same
Go version. `prepare-ast.sh` builds the shared syntax nodes and Go printer from
that version. `make prepare` calls these scripts. See
[checker preparation](../internal/checker/upstream.md) for inputs, generated
files, and Go updates.
See [frontend preparation](../internal/frontend/upstream.md) for the grammar
patch and parser updates.

`check-source-size.go` limits each tracked Go or TGo source file to 700 lines.
It uses `git ls-files`, reads the listed files, and reports each path that exceeds
the limit. Comments and blank lines count. A final line without a newline also
counts; an empty file has zero lines.

The size check includes tests, fixture inputs, and reference programs. It skips
names ending exactly in `_tgo.go` or `_tgo_test.go`. Untracked generated checker
source is not in the Git file list.

`check-dead-code.sh` uses the pinned Go `deadcode` tool to find functions that
no entry point can reach. It checks the repository with tests included, then
checks public functions under `pkg/` from `cmd/tgo` without tests. It fails on
any finding. The second check catches public APIs kept alive only by tests.

`check-function-comments` requires an attached comment on every named Go
function and method, including private helpers, tests, benchmarks, `main`, and
`init`. It checks tracked handwritten files. It skips fixture programs under
`tests/testdata`, generated Go files, and `_tgo.go` and `_tgo_test.go` outputs.
Anonymous functions have no declaration name and are outside this rule. Empty
comments and tool directives alone do not satisfy it.

`make lint` prepares the compiler sources, runs the source-size, function-comment, and dead-code
checks, runs Go linters,
and checks Go formatting. The pull request workflow runs the same command.
See [source checks](../docs/guide/lint.md) for tool versions and limits.
