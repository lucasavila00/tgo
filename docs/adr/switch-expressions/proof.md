# Switch expressions: PROOF

## Behavior checks

Extend `tests/testdata/value-expressions/` with switch fixtures, snapshots,
expected output, and independent Go references. Compile and run both programs;
compare values and event logs.

Check initializer-before-tag order, one tag evaluation, ordered cases, and
skipped cases after a match. A default written first must wait for all cases
to fail. For `200`, the WHAT example audits once and produces `"ok"`.

Check expressionless and type switches, clause scopes, nested expressions,
short-circuit operands, loop visits, stores, and package initialization.
Returns skip later work; defers run at function exit; outer jumps retain
targets. Panics retain value and order. `!` preserves error identity; `!!` adds
context while preserving `errors.Is` and `errors.As` access.

## Rejected inputs

Require source-position diagnostics for missing default or enum coverage,
missing values, incompatible types, multiple results, constant-only uses,
escaped bindings, value-switch breaks, and fallthrough. Complete enum coverage
succeeds; missing forms and duplicate cases fail.

## Regression checks

Check ordinary-switch breaks and fallthrough, nested-loop breaks, contextual
types, nil, constants, generic inference, and runtime overflow.

## Evidence

`TestValueExpressions` includes `Switch`, `SwitchTypes`, and `SwitchControl`.
Each input has a generated snapshot, expected output, and an independent Go
reference. Run `go test ./tests -run '^TestValueExpressions$'`.

The source checker tests accepted forms and rejected inputs, including source
positions. The parser tests nested switches and source positions. Existing
if-expression and lowering tests check regression behavior.

Switch expressions with `default` are implemented. Enum declarations are not
implemented. Verified enum coverage with `exhaustive:` remains a dependency;
this implementation requires `default`.
