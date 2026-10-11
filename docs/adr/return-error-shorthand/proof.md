# Short returns: PROOF

## Behavior checks

Add fixtures under `tests/testdata/return-error-shorthand/`, with generated
Go snapshots and independent Go references. Require identical values, error
identity, and call order.

Check all three forms, generic zeros, multiple-result calls, and typed nil
errors. In `(int, error)`, `return .. err` must return `0, err`. Test
propagation, operand returns, panics, defers, and jumps: reached operands
run once; early exits skip later operands.

## Rejected inputs

Reject both forms with named results, including `_` and grouped names.
Reject bare `return ..`, both operator positions together, wrong result
counts, incompatible operands, final non-error types, and nil incompatible
with the final type. Diagnostics must identify the original `.tgo` position.

## Regression checks

Cover contextual constants, shifts, nested functions, comments, multiline
syntax, floats, selectors, `...`, and shadowed `nil`, `error`, and type
names. Check termination analysis and ordinary returns with named results.

Compare automatic fixes before and after execution. Leave named-result
functions, effectful calls, non-nil collections, nonzero interface
conversions, and unproved values unchanged.

## Evidence

These feature checks are planned. Add `TestReturnErrorShorthand` and run:

```sh
go test ./tests -run '^TestReturnErrorShorthand$'
```

A passing test must verify output compilation, reference behavior,
snapshots, and diagnostics.

[WHAT](what.md) states behavior. [HOW](how.md) explains lowering.
