# Sum types: PROOF

## Behavior checks

Add `tests/testdata/sum-types/` inputs, generated snapshots, and handwritten Go
references. Compare values, effects, and errors. Check all zero sources, empty
variants, repeated field types, named types, generics, and indirect recursion.
Check zero numeric, string, pointer, and struct payload fields. Reorder variants;
zero must select the new first variant.

Check matching after alias writes and calls: the copied payload stays unchanged;
referenced data stays shared. Test constructor operand order, whole-value
replacement, inactive pointer clearing, and garbage collection. Check Go
constructors and accessors across packages, including zero enums. The first
variant's accessor must succeed.

Accept partial matches with `default:` and complete matches with `exhaustive:`.

## Rejected inputs

Require source-position errors for empty enums, missing exhaustive cases,
duplicate cases, value switches with neither label, unproved payload access,
and direct infinite-size recursion.

## Regression checks

Preserve ordinary switches. Check short returns and
propagation: zero enum results select the first variant; error identity stays unchanged.

## Evidence

Run `go test ./tests -run '^TestSumTypes$' -bench '^BenchmarkSumTypes$' -benchmem`.

Require the HOW sizes on amd64 and zero added allocations for string
construction/copy/read. Check tag widths: 256 variants fit `uint8`; 257 need
`uint16`. Check larger boundaries, GC retention, and other architectures.
