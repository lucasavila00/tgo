# PROOF

Extend value-expression fixtures with TGo inputs, generated Go snapshots,
expected output, and independent handwritten Go references. Compile and run both
implementations. Compare values and event logs, not only generated text.

Verify:

- Initializer before tag; tag once; case expressions left to right, including
  multiple expressions per clause; later cases skipped after a match.
- Default written first still waits for all cases to fail. Expressionless
  switches select the first true case.
- Type switches retain single-type, multiple-type, nil, and default bindings
  and scopes.
- Nested if/switch values preserve calls, short-circuit operands, loop visits,
  assignment destinations, and package initialization.
- Returns skip later work; defers run at function exit; outer jumps retain
  targets; nested-loop breaks remain valid; panics retain value and order.
- `!` preserves error identity. `!!` adds call context; `errors.Is` and
  `errors.As` still find the cause.
- Contextual and named types, nil, constants, generic inference, and runtime
  overflow follow if-expression rules.
- Complete enum coverage succeeds; missing forms and duplicate cases fail.

Require source-position diagnostics for missing default or coverage, missing
values, incompatible types, multiple results, constant-only uses, escaped clause
bindings, value-switch breaks, and fallthrough. Include valid ordinary switches
with break and fallthrough to detect regressions.
