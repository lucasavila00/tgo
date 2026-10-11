# PROOF

Add table-driven integration fixtures under
`tests/testdata/value-expressions/`. Each directory contains `input.tgo`,
`output.go` (the reviewed generated-code snapshot), handwritten `reference.go`,
and `expect.json` (results, ordered effects, and expected errors).

| Directory | Required cases |
| --- | --- |
| `if` | Both branches, nested `else if`, initializer scope |
| `ordering` | Calls, operators, literals, assignment targets, skipped `&&`/`||` |
| `branches` | Local bindings, shadowing, final calls, nested expressions |
| `types` | Named types, interfaces, constants, nil, generics, contextual types |
| `control` | Return, panic, all branches terminate, loop continue, labels, goto, enclosing break |
| `errors` | `!` wrapping, `!!` identity, skipped branches, deferred calls |
| `invalid` | Missing else/value, mixed types, tuple result |
| `statements` | Unchanged ordinary Go if behavior |

`TestValueExpressions` loops over every directory. It compiles the TGo source,
formats the result, and compares it with `output.go`. Compile and run the
result and the handwritten Go reference in separate test modules. Require the
same output, side-effect trace, panic, and defer order. For error identity,
assert equality with the source sentinel inside each program.

Use multiple inputs per fixture to reach every branch and skipped path. Put
observable effects in headers, earlier operands, branch
statements, and later operands. A snapshot alone cannot prove evaluation order.

`TestValueExpressionDiagnostics` checks each invalid source against expected
file, line, column, and diagnostic text. Add parser cases with comments,
newlines, nested literals, and adjacent ordinary statements. Check branch-local
names cannot escape and generated names cannot capture source names.

Compile every valid generated snapshot with the pinned Go version. Run each
compilation twice and compare bytes for deterministic output. Fixture changes
require explicit snapshot review; tests must not update snapshots on failure.

Run focused checks with:

```sh
go test ./tests -run '^TestValueExpression' -count=1
```

Run the full test suite in GitHub Actions. These tests are required for the
implementation; this ADR does not claim they already exist.

[WHAT](what.md) defines behavior. [HOW](how.md) defines implementation.
