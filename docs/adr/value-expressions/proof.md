# PROOF

Use TGo inputs, Go snapshots, handwritten Go references, and expected results
in `tests/testdata/value-expressions/`.

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
and references. Compare values, effects, panic, and defers.
Assert sentinel identity. Exercise both branches and skipped paths. Compare
repeated compilation bytes.

Check diagnostic positions and text:
missing else/value, mixed types, tuples, constant-only contexts, escaped locals,
unknown types, and types that generated declarations cannot name. Reject
`continue` to a range inside its collection; keep outer-loop and body jumps.

Review snapshots. Never update them on failure.
