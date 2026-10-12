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

## Dead code

`make lint` also runs Go's `deadcode` tool from the pinned `golang.org/x/tools`
module version. First, it checks functions across the repository with test
entry points included. Then, it checks functions in `pkg/` using only the
`tgo` command as an entry point. Any reported function fails the check.

The second check excludes tests. A test can call an obsolete public API and
keep it alive even when the compiler does not use it. Internal lowering helpers
can remain available for proof tests. The existing `unused` linter also checks
unused declarations, including types and variables.

## Compiler dependencies

Depguard checks imports in the existing lint job. It also checks test files
inside compiler packages. The rules follow the package responsibilities in the
[driver](../../internal/driver/readme.md),
[checker](../../internal/checker/readme.md),
[frontend](../../internal/frontend/readme.md), and
[lowering](../../internal/lowering/readme.md) documentation.

The current dependency direction is:

| Package | Compiler packages it uses |
| --- | --- |
| `cmd/tgo` | `pkg/transpiler` |
| `pkg/transpiler` | `internal/driver` |
| `internal/driver` | `internal/checker`, `internal/lowering` |
| `internal/lowering` | `internal/checker`, `internal/frontend`, lowering helpers |
| `internal/lowering/source` | `internal/checker` |
| `internal/checker` | `internal/frontend`, private checker packages |
| `internal/checker/types` | `internal/controlflow`, private upstream packages |
| `internal/frontend`, `internal/controlflow`, `internal/lowering/names` | None |

All internal packages must stay independent of the public API and command.
Compiler packages below the driver must not import the driver. The frontend
and control flow analysis must not import the checker or lowering. The checker
must not import lowering. The private checker and its generated upstream
packages follow these same rules. These rules do not limit standard library
imports or external dependencies.

## Run the checks

Where local validation is permitted, install the pinned tool and use:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
make lint
```

Use hosted CI for full compiler validation. See the
[test guide](../../tests/readme.md) for the remaining workflow steps and
[the scripts](../../scripts/README.md) for the file-size check.
