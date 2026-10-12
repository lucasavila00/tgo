# Source checks in CI

The pull request workflow runs `make lint` before documentation and tests. It
prepares the private checker, checks source-file size, runs Go lint checks, and
checks formatting.

## Tools and limits

The repository pins golangci-lint v2.14.0. Its checks cover unchecked errors,
Go vet findings, Staticcheck bugs, unused code, documentation, and basic names.
Revive supplies style checks. The formatter checks Go formatting.

Go source lines are limited to 120 characters. Split long calls, conditions,
and literals at clear boundaries.

Each function has a cognitive complexity limit of 20. This check counts nested
branches and loops. Split complex work into named steps.

Each function is limited to 100 lines, including comments. Each tracked Go or
TGo source file is limited to 700 lines, including comments and blank lines.
Split work into steps with clear inputs and results when a limit is exceeded.

The file-size check includes fixture inputs and reference programs. It skips
names ending exactly in `_tgo.go` or `_tgo_test.go`. Go lint and formatting exclude
`tests/testdata`; normal test files outside that directory remain checked.

## Run the checks

Where local validation is permitted, install the pinned tool and use:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
make lint
```

Use hosted CI for full compiler validation. See the
[test guide](../../tests/readme.md) for the remaining workflow steps and
[the scripts](../../scripts/README.md) for the file-size check.
