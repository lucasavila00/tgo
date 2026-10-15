# Sum types: PROOF

## Behavior checks

Add `tests/testdata/sum-types/` inputs, generated snapshots, and handwritten Go
references. Compare values, effects, and errors. Check the WHAT example, all zero
sources, nil comparisons, empty variants, repeated field types, named types,
generics, and indirect recursion.

Check matching after alias writes and calls: the copied payload stays unchanged;
referenced data stays shared. Test constructor operand order, whole-value
replacement, inactive pointer clearing, garbage collection, and payload contracts
for non-null pointers and checked values. Check generated Go constructors and
successful/failed accessors across packages.

## Rejected inputs

Require source-position errors for missing nil/variant cases, duplicate cases,
unproved payload access, invalid payload construction, and direct infinite-size
recursion.

## Regression checks

Keep ordinary switches and value switches unchanged. Check short returns and
propagation: enum results become nil and error identity stays unchanged.

## Evidence

Add `TestSumTypes` and `BenchmarkSumTypes` under `tests/`. Run
`go test ./tests -run '^TestSumTypes$'` and
`go test ./tests -run '^$' -bench '^BenchmarkSumTypes$' -benchmem`.

Require the HOW sizes on amd64 and zero added allocations for string
construction/copy/read. Check tag-width boundaries, GC retention, and layouts
on other architectures. Compare allocation counts with handwritten Go references.
These are required checks for implementation.
