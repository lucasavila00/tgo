# Lowering packages: PROOF

## Behavior checks

Use [propagation_test.go](../../../tests/propagation_test.go),
[value_expressions_test.go](../../../tests/value_expressions_test.go),
[comprehensions_test.go](../../../tests/comprehensions_test.go),
[short_returns_test.go](../../../tests/short_returns_test.go), and
[insert_test.go](../../../tests/insert_test.go).
The [propagation reference](../../../tests/testdata/propagation-reference) and
[proof corpus](../../../tests/testdata/proof) supply independent behavior checks.
Require unchanged traces, values, errors, panics, and deferred calls.

## Rejected inputs

No new restrictions. Preserve existing diagnostics in
[propagation-rejected](../../../tests/testdata/propagation-rejected) and
[comprehensions-rejected](../../../tests/testdata/comprehensions-rejected).

## Regression checks

Preserve snapshots, contextual and tuple types, assignment order, skipped paths,
loop posts, labeled jumps, closures, initializer state, and error identity.
Callbacks must receive the current context and preserve nested return targets.

## Evidence

Planned focused commands, from the repository root:

```sh
go test ./internal/lowering
go test ./tests -run '^(TestPropagationFixtures|TestValueExpressions|TestCollectionComprehensions|TestReturnErrorShorthand|TestInsertReturn)$'
go list -deps ./internal/lowering/...
make lint
make docs-check
```

Inspect the dependency list: shared must not import rules. Compare snapshots
and public APIs. Use hosted CI for the full compiler suite.

Completed: baseline `go test ./internal/lowering` passed before extraction.
ADR word limits, local links, and diff checks passed. The code move and its
behavior checks remain future work under #347; this PR contains only the ADR.
