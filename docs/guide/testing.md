# Integration tests

These tests use the public `pkg/transpiler` API. Compiler unit tests stay
beside their implementation in `internal/`; the public API boundary test stays
in `pkg/transpiler`.

## Propagation fixtures

`TestPropagationFixtures` loops over `tests/testdata/propagation/`. Each directory has:

- `.tgo` sources, a `go.mod`, and any Go dependencies or assets.
- Generated `_tgo.go` snapshots beside each selected source.
  Test and target suffixes follow `_tgo`, as in `name_tgo_linux_amd64_test.go`.
- `stdout.txt`: the expected program output.

The sources drive compilation; existing generated files are ignored.
The runner compares generated file bytes with the committed snapshots. It
then runs the generated program and checks its output. Generated test files
also run with `go test`. Cases cover both propagation operators, evaluation
order, conditional execution, storage, generic types and private-type diagnostics, error
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
The proof corpus in `tests/testdata/proof/` tests insertion after each expression,
including skipped paths and later loop visits. Its handwritten reference traces
and errors check behavior independently of generated-code snapshots.

Run focused checks locally. GitHub Actions runs the full suite with
`go test ./...`. See the [proof plan](../adr/expression-lowering/proof.md)
for the expression inventory and behavior checks.

GitHub Actions has a fixture-build step that compiles these inputs and checks
their snapshots before the full suite. It does not update committed files.

`tests/testdata/proof/rejections.json` lists the exact expression sites that require
an inaccessible type in a generated typed temporary. Proof tests require the
specified error at those sites and retain all sites in `manifest.json`.
`tests/testdata/typed-temporaries-rejected/` contains TGo inputs and exact diagnostics
for private types and hidden names.
