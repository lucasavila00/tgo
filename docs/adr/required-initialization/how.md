# HOW

Check source before expression lowering. Generated locals are outside these
checks.

Use `go/types.Info` to resolve literal types, aliases, fields, and constant
indices. Inspect each source `ValueSpec` for initializer expressions. Let Go
check assignment counts and types.

For struct literals, compare supplied field objects with the underlying
struct's direct fields. Go already checks positional literal completeness.
For arrays and slices, calculate explicit and implied indices and the resulting
length; compare supplied indices with that length. Apply the same check to the
underlying literal type of a type parameter. Do not require map key coverage.

Inspect `go/ast.FuncType.Results`. Reject result fields with a nonempty `Names`
list. This also finds nested function types, aliases, and interface methods.
Report errors at source declarations. Do not inspect result names in imported
`go/types.Signature` objects.

Let Go check return operands, arity, types, and missing returns. No control-flow
or assignment-state analysis is needed. Emit normal Go; retain existing
failure-return and propagation lowering.

[Go specification](https://go.dev/ref/spec)
