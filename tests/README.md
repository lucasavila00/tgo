# Integration tests

These tests use the public `pkg/transpiler` API. Compiler unit tests stay
beside their implementation in `internal/`; the public API boundary test stays
in `pkg/transpiler`.

## Propagation fixtures

`TestPropagationFixtures` loops over `testdata/propagation/`. Each directory has:

- `input/`: a Go module with TGo sources and any Go dependencies or assets.
- `output/`: committed snapshots of every emitted file, including generated Go.
- `stdout.txt`: the expected program output.

The runner compares the emitted file names and bytes with the snapshots. It
then runs the generated program and checks its output. Generated test files
also run with `go test`. Cases cover both propagation operators, evaluation
order, conditional execution, storage, generic and private types, error
identity, build selection, assets, and test packages.

Run one case from the repository root:

```sh
go test ./tests -run '^TestPropagationFixtures/Calls$' -count=1
```

After an intended output change, regenerate the affected snapshot and inspect
the diff:

```sh
go test ./tests -run '^TestPropagationFixtures/Calls$' -update-snapshots -count=1
```

The update flag changes snapshots only. Expected runtime output stays handwritten.

## Expression insertion

Insertion tests check effects, control flow, storage, and return boundaries.
The proof corpus in `testdata/proof/` tests insertion after each expression,
including skipped paths and later loop visits. Its handwritten reference traces
and errors check behavior independently of generated-code snapshots.

Run focused checks locally. GitHub Actions runs the full suite with
`go test ./...`. See the [proof plan](../docs/adr/expression-lowering/proof.md)
for the expression inventory and behavior checks.
