# Compiler integration tests

These tests call `pkg/transpiler` to compile complete modules. They check final
Go files, original diagnostic positions, and runtime behavior. Unit tests stay
beside their implementation in `internal/`; API tests stay in `pkg/transpiler`.

## Fixtures

Propagation inputs live in `testdata/propagation/`. If-value inputs live in
`testdata/value-expressions/`. TGo source controls compilation. Matching
`_tgo.go` files are expected snapshots, not source input.

The runner reads committed expectations and copies each fixture into a test
directory. Compilation writes there. Tests read the generated files from disk,
compare snapshots, then run the program and check handwritten output. Where
present, separate Go references run against the same expected output. Generated
test files also run with `go test`.

The compiler does not create these fixture copies. Tests create them to protect
committed files from in-place compilation.

## Behavior and rejection checks

The expression proof tests insert neutral markers and early returns at recorded
sites. Handwritten expectations check traces, values, assignments, panics, jumps,
and deferred calls. Separate reference programs check propagation behavior.
Rejection fixtures require specified diagnostics for invalid source and types
that generated declarations cannot name.

Module tests check stale snapshots, build selection, subdirectory arguments,
embedded assets, cgo headers, vendor and workspace dependencies, replacements,
module updates, and retry after a failed source stage.

The separate `loweringfuzz` tag enables propagation and if-value fuzz campaigns.
Both compare generated programs with independent Go references.

See the [test guide](../docs/guide/testing.md) for fixture commands, snapshot
updates, the proof inventory, and CI. See [fuzzing](fuzzing.md) for campaign
coverage and failure investigation.
