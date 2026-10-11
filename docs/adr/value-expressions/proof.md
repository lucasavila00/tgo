# Check if-value behavior

Use TGo inputs, generated snapshots, handwritten Go references, and expected
outputs in `tests/testdata/value-expressions/`.

Cover:

- Selected branches, nested else-if, header initializers, and ordinary if statements.
- Calls, operators, literals, assignment targets, and skipped Boolean operands.
- Branch locals, outer names before declarations, final calls, and nested values.
- Named types, interfaces, constants, nil, generics, surrounding types, and shifts.
- Returns, panic, all branches terminating, continue, break, labels, and goto.
- Propagation identity and wrapping, skipped calls, and deferred execution.
- Package dependencies, initialization order, and compiled imports.
- Comments, newlines, source positions, and generated-name conflicts.

Generic cases must include several conditional arguments and inference from
other arguments. Contextual integer cases must preserve runtime overflow:
`(if ok {127} else {0}) + 1` in an `int8` context is a runtime value, not a
constant-overflow rejection.

Run compiled fixtures and separate Go references. Compare values, traces,
panics, defers, and sentinel error identity. Check stable repeated output.

Rejection tests check missing else or value, incompatible branch types, tuples,
constant-only uses, escaped locals, unknown types, and inaccessible generated
types. Check exact source positions where specified. Reject continue to a range
inside its collection while retaining valid body and outer-loop jumps.

Tagged if-expression fuzz tests provide additional bounded behavior checks.
Review snapshot changes; never update expected results just to accept a failure.

[WHAT](what.md) defines the rules. [HOW](how.md) describes the implementation.
