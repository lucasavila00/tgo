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

Completed: a temporary Go prototype checked amd64 sizes (24/40/24/16 bytes), alignment
(8), zero allocations for string construction/copy/read, scalar packing, copied tags, live
references after GC, and inactive pointer clearing. Command:
`go test -v /tmp/tgo-sum-layout-proof/layout_test.go`.
These measurements cover generated-style Go, not a compiler implementation.

Planned: focused compiler fixtures, allocation benchmarks, race checks, tag-width
boundaries, and full hosted CI. GC retention and cross-architecture measurements
remain untested.
