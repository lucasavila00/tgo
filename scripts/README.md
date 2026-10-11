# Source checks

`check-source-size.go` limits tracked Go and TGo source files to 700 lines.
The check includes tests and fixture inputs. It reports the path and line count
for each file that exceeds the limit. An empty file has zero lines. A final line
without a newline still counts as one line.

The check excludes compiler outputs named `*_tgo.go` and `*_tgo_test.go`.
These files come from fixture inputs and can exceed the limit after lowering.
The check uses `git ls-files`, so build files that are not tracked are excluded.

Run `make lint` from the repository root. This runs the size check, Go linters,
and the Go format check. Pull request CI runs the same command.
