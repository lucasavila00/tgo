# Compiler tests

The compiler must produce valid Go and preserve TGo execution. These are
separate requirements. A snapshot can show the emitted code, but a handwritten
reference is needed to check its behavior independently.

## Unit tests and integration tests

Tests beside `internal/` code check parsing, type rules, control flow, lowering,
and the driver. Tests in `pkg/transpiler` check the public API. Tests in `tests/`
use that API to compile complete modules and run the results.

The driver tests also check original disk paths: embedded assets, cgo headers,
vendor files, workspace dependencies, relative replacements, excluded files,
and retry after a source-checking failure.

## Fixture inputs, snapshots, and expected output

Propagation fixtures live in `tests/testdata/propagation/`. If-value fixtures
live in `tests/testdata/value-expressions/`. Collection comprehensions live in
`tests/testdata/comprehensions/`. A fixture contains:

- `.tgo` inputs and a `go.mod`, with any ordinary Go source or assets.
- `_tgo.go` snapshots beside selected inputs, with test and platform suffixes
  at the end.
- `stdout.txt`, written by hand to state the expected program output.
- A separate Go reference where that fixture needs an independent implementation.

The runner reads committed snapshots before compilation, then copies the
fixture into a test directory. The compiler writes in that directory. The runner
reads generated files from disk, compares them with snapshots, and runs the
program. Generated test files also run with `go test`.

The compiler itself creates no temporary module. Fixture copies protect the
committed snapshots from its in-place writes. Existing snapshots never supply
source input.

Run these commands from the repository root. For a focused fixture check:

```sh
go test ./tests -run '^TestPropagationFixtures/Calls$' -count=1
```

After an intended output change, update only the affected snapshots and inspect
the diff:

```sh
go test ./tests -run '^TestPropagationFixtures/Calls$' -update-snapshots -count=1
```

The flag updates generated Go snapshots. It does not change handwritten expected
output. Do not accept a behavior change by updating snapshots alone.

## Expression insertion proof

`tests/testdata/proof/` records expression sites in `manifest.json`. For eligible
sites, the tests insert either a neutral trace marker or an early return. They
check reached paths, skipped paths, and later visits inside loops.

Handwritten `reference.go` and `expect.json` define expected traces and results.
Checks include assignment storage, error identity, panics, jumps, and defers.
`rejections.json` records sites where a required saved type cannot be named in
generated Go. Those sites remain in the inventory and must report their specified
error. See [the proof plan](../docs/adr/expression-lowering.md).

`tests/testdata/typed-temporaries-rejected/` has TGo examples with exact diagnostics
for inaccessible types and hidden names. Other checker and rejection tests cover
invalid propagation returns and invalid if-value types and control flow.

## Fuzz campaigns

The separate `loweringfuzz` tag enables propagation and if-value fuzzers. Each
compares generated programs with independent Go references. Normal CI does not
run these campaigns. See [the fuzz guide](fuzzing.md) for commands,
coverage, and how to investigate a failure.

## CI validation

The pull request workflow runs these steps in order:

1. `make lint`: source-size limits, Go lint checks, and formatting.
2. `make docs-check`: compare generated package READMEs with Go comments.
3. A fixture step: compile propagation inputs and compare their snapshots.
4. `go test ./...`: run the full normal test suite.

Build preparation generates the private checker before these checks. Use CI for
the full suite. A successful final Go check is one part of correctness; the
runtime references and tests check the source behavior it cannot prove.
