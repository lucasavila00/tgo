# Sum types: PROOF

## Behavior checks

Add `tests/testdata/sum-types/` fixtures and snapshots. Compare results and call
order with handwritten Go.

- Zero `Account` produces `Personal{Name: ""}`. `Account.Business{Company: "Acme"}` produces
  `"Acme"`. Reordering changes zero.
- Missing map keys, closed channels, generic zeros, and error returns select
  the first variant. Check pointer and struct fields.
- A zero boxed first variant returns zero fields without a panic.
- Payload copies survive enum replacement. Copied pointers share data.
- Constructors clear inactive storage; live references survive garbage collection.
- Go constructors, `Tag()`, and payload methods work across packages.
- Partial matches with `default:` and complete matches with `exhaustive:` pass.

## Rejected inputs

An incomplete exhaustive match must name missing variants, such as `Business`. Added
variants must make incomplete exhaustive matches fail. Reject empty enums,
duplicate cases, value switches with neither label, payload fields in
grouped/default cases, and direct infinite-size recursion. Report source positions.

## Regression checks

Preserve ordinary switches, Go interface type switches, and error identity.

## Evidence

Run `go test ./tests -run TestSumTypes -bench BenchmarkSumTypes -benchmem`.
Compare allocations with references. On amd64, `[79]byte` stays inline at
80 bytes; `[80]byte` boxes to 24. Test size ties, nested enums, and 256/257
variants selecting `uint8`/`uint16`.
