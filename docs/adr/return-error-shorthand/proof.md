# Short returns: PROOF

## Behavior checks

Add fixtures under `tests/testdata/return-error-shorthand/`, with generated
Go snapshots and independent Go references. Require identical values, error
identity, and call order.

Check all three forms, generic zeros, multiple-result calls, typed nil
errors, and named results. The deferred increment in [WHAT](what.md) must
return `1, nil`. Test propagation, operand returns, panics, and jumps:
reached operands run once; early exits skip later operands and outer result
assignment.

## Rejected inputs

Reject bare `return ..`, both operator positions together, wrong result
counts, incompatible operands, final non-error types, and nil incompatible
with the final type. Diagnostics must identify the original `.tgo` position.

## Regression checks

Cover contextual constants, shifts, nested functions, comments, multiline
syntax, floats, selectors, `...`, and shadowed `nil`, `error`, and type
names. Check termination analysis and ordinary returns.

Compare automatic fixes before and after execution. Leave effectful calls,
non-nil collections, nonzero interface conversions, and unproved values
unchanged.

## Evidence

These feature checks are planned. Add `TestReturnErrorShorthand` and run:

```sh
go test ./tests -run '^TestReturnErrorShorthand$'
```

A passing test must verify output compilation, reference behavior,
snapshots, and diagnostics. Documentation review alone cannot establish
these results.

[HOW](how.md) explains lowering.
