# Compiler preparation and source checks

`prepare-checker.sh` builds private Go checker source from a pinned version and
patch. `make prepare` calls it. See
[checker preparation](../internal/checker/upstream.md) for inputs, generated
files, and Go updates.

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

`make lint` prepares the checker, runs the source-size and dead-code checks, runs Go linters,
and checks Go formatting. The pull request workflow runs the same command.
See [source checks](../docs/guide/lint.md) for tool versions and limits.
