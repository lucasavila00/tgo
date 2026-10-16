# Sum types: PROOF

## Behavior

[Fixtures](../../../tests/testdata/sum-types/) contain TGo inputs, generated Go,
expected output, and independent Go programs. Compare values and call order.

Check first-variant zeros from declarations, maps, channels, generics, and error
returns. Check payload copies, shared pointers, replacement, nested enums, and
constructors called from Go and other TGo packages.

Check complete matches, default cases, grouped cases, aliases, and one input
evaluation. Preserve returns, defers, jumps, and error identity inside cases.

## Invalid inputs

[Diagnostic fixtures](../../../tests/testdata/sum-types-rejected/) check missing
variants, duplicate cases, empty enums, invalid payload fields, and recursive
values with infinite size. Errors must point to the source. Checker tests also
check that generic operations cannot expose private enum storage.

## Commands

```sh
go test ./tests -run '^TestSumType'
go test ./internal/checker/types -run '^TestEnum'
go -C tests/testdata/sum-types/Allocations test -run '^$' -bench BenchmarkSumTypes -benchmem
```

The benchmark runs in the `Allocations` fixture. Compare its results with its
Go reference after the storage decision. Add checks for the
chosen 80-byte rule, tag widths, equality, negative zero, map keys, and allocation counts
before accepting that layout.
