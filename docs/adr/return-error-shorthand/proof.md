# Check return values and effects

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

Add `.tgo` fixtures, generated Go snapshots, and independent Go references under
`tests/testdata/`. Compile and run both programs against expected values and
traces: records of calls and defers in execution order. References use explicit
Go returns.

Check:

- All three examples, error-only returns, multiple-result calls, every
  zero-value kind, named types, and generic results.
- Named results with previous values, grouped and blank names, shadowed names,
  and defers that read or change results.
- Concrete and generic errors, richer interfaces, and typed nil conversions.
  Compare error identity, `errors.Is`, and `errors.As`.
- Contextual constants, shifts, nil, if values, and nested functions.
- Propagation, operand returns, panics, and jumps. Reached operands run once;
  early exits skip later operands and outer result assignment.
- Comments, multiline syntax, floats, selectors, `...`, ordinary returns,
  original positions, and shadowed `nil`, `error`, and type names.

Require diagnostics for wrong counts, incompatible types, missing operands,
and invalid operator positions. Check termination analysis.

For automatic fixes, compare execution before and after accepted changes.
Leave effectful calls, non-nil collections, nonzero interface conversions,
and unproved variable values unchanged.

[WHAT](what.md) states behavior. [HOW](how.md) explains lowering.
