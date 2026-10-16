# Sum types: PROOF

## Behavior checks

Add `tests/testdata/sum-types/` inputs, generated snapshots, and handwritten Go
references. Compare values, effects, and errors. Check the example, zero
sources, empty variants, repeated field types, named types, generics, and
indirect recursion. Give the first variant numeric, string, pointer, and nested
struct fields. Check their zero values. Change variant order; zero must select
the new first variant.

Check matching after alias writes and calls: the copied payload stays unchanged;
referenced data stays shared. Test constructor operand order, whole-value
replacement, inactive pointer clearing, and garbage collection. Check Go
constructors and accessors across packages, including the first variant's
accessor on a zero enum.

## Rejected inputs

Require source-position errors for empty enum declarations, missing or duplicate
variant cases, unproved payload access, and direct infinite-size recursion.

## Regression checks

Preserve ordinary switches. Check short returns and
propagation: zero enum results select the first variant; error identity stays unchanged.

## Evidence

Run
`go test ./tests -run '^TestSumTypes$'` and
`go test ./tests -run '^$' -bench '^BenchmarkSumTypes$' -benchmem`.

Require the HOW sizes on amd64 and zero added allocations for string
construction/copy/read. Check tag widths: 256 variants fit `uint8`; 257 need
`uint16`. Check larger boundaries, GC retention, and other architectures.
