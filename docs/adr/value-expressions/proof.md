# PROOF

Add `tests/testdata/value-expressions/` fixtures with TGo inputs, Go snapshots,
handwritten Go references, and expected results. These checks are planned.

Cover:

- Both branches, nested `else if`, initializer scope, ordinary `if` statements.
- Calls, operators, literals, assignment targets, skipped `&&` and `||` operands.
- Branch locals, outer names before `:=`, final calls, nested expressions.
- Package dependencies, declaration order, selected effects, compiled imports.
- Named types, interfaces, constants, nil, generics, contextual types, shifts.
- Return, panic, all branches terminating, continue, break, labels, goto.
- `!` identity, `!!` error text, skipped propagation, deferred calls.
- Comments, newlines, nested literals, adjacent statements, generated-name collisions.
- Generic calls with several conditional arguments; public `Load` stock-package identity.
- Runtime `int8` overflow (-128 when `ok=true`) in int8 context: `(if ok {127} else {0})+1`, without false constant-overflow errors.

`TestValueExpressions` compiles inputs and compares snapshots. Run generated Go
and references separately. Compare results, effects, panic, and defer order.
Assert sentinel identity. Exercise both branches and skipped paths. Compare
repeated compilation bytes.

`TestValueExpressionDiagnostics` checks diagnostic file, line, column, and text:
missing else/value, mixed types, tuples, constant-only contexts, escaped locals,
unknown types, and types that generated declarations cannot name.

Review snapshots; tests must not update them on failure.
