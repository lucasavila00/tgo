# Short returns: PROOF

## Behavior checks

`TestReturnErrorShorthand` compiles `.tgo` fixtures, checks committed Go snapshots,
and runs output and independent Go references against expected output.

Eight groups cover all three forms, every zero-value category, generic and
concrete error types, typed nil conversions, tuple calls, contextual shifts,
operand order, and propagation. Control cases cover defers, panics, jumps,
early returns, and skipped operands. Function literals have their own return
signatures, including inside package initializers and unevaluated operands.

## Rejected inputs

Rejection fixtures check named results, including `_` and grouped names;
wrong counts and types; missing error results; invalid nil conversions;
overflow; and invalid syntax. Diagnostics must identify the original TGo
source. Checker tests verify exact positions for named results and overflow.

## Regression checks

Cover ordinary named-result returns, nested functions, comments, multiline
syntax, floats, variadic calls, and shadowed names. Generated names and types
must keep the same source bindings.

Automatic modernization requires proof that omitted values and effects stay
unchanged.

## Evidence

```sh
go test ./tests -run '^TestReturnErrorShorthand$'
```

The focused fixture and checker tests pass. [HOW](how.md) explains lowering.
