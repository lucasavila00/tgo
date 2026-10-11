# PROOF

Add fixtures under `tests/testdata/value-expressions/`. Each contains TGo input,
a generated Go snapshot, handwritten Go reference, and expected results.
These are implementation checks to add.

Cover:

- Both branches, nested `else if`, initializer scope, ordinary `if` statements.
- Calls, operators, literals, assignment targets, skipped `&&` and `||` operands.
- Branch locals, outer names before `:=`, final calls, nested expressions.
- Package dependencies, declaration order, selected branch effects.
- Named types, interfaces, constants, nil, generics, contextual types.
- Return, panic, all branches terminating, continue, break, labels, goto.
- `!` error text, `!!` error identity, skipped propagation, deferred calls.
- Comments, newlines, nested literals, adjacent statements, generated-name collisions.

`TestValueExpressions` compiles each input and compares its snapshot. Compile
and run generated Go and the reference separately. Compare results, ordered
effects, panic, and defer order. Assert sentinel identity inside each program.
Use inputs that reach both branches and skipped paths. Repeat compilation and
compare bytes.

`TestValueExpressionDiagnostics` checks file, line, column, and text for missing
else/value, mixed types, tuple results, constant-only contexts, escaped branch
locals, unknown result types, and types that generated declarations cannot name.

Run:

```sh
go test ./tests -run '^TestValueExpression' -count=1
```

Review snapshots explicitly; tests must not update them on failure.
