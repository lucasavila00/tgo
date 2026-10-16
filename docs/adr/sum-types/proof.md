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
go test ./internal/checker/types -run 'TestEnum|TestNestedEnumLayout|TestLocalEnumDeclaration'
go -C tests/testdata/sum-types/Allocations test -run '^$' -bench BenchmarkSumTypes -benchmem
```

Check both sides of the 80-byte and 256-variant boundaries. Keep large first
payloads inline. Check largest-first ties, nested sizes, and boxing overhead.
Compare zero values and constructed values as map keys. Preserve floating-point
negative zero. Boxed noncomparable payloads must remain noncomparable.

The benchmark compares allocation counts with ordinary Go value boxing.
