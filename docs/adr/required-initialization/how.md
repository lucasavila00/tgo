# HOW

Implement this after approval in `compilerv2`. Do not restore the old compiler
or specification. Apply checks to source objects before expression lowering;
generated Go locals are not source declarations.

Use `go/types.Info` to identify declarations, literal types, field objects,
constant indices, and assignments. Require values in each source `ValueSpec`.
Let Go check assignment counts and types. For `:=`, check only newly declared
objects; Go's assignment rules still apply to existing objects.

For struct literals, compare supplied fields with all direct fields of the
underlying struct. Positional literals already require all fields in Go.
Resolve aliases and embedded fields by object identity. For array and slice
literals, compute each explicit or implied index and the resulting length.
Require distinct supplied indices to cover that length. An empty slice has
length zero. For a type parameter, use the underlying literal type required
by Go; apply the same rule. Do not inspect map keys for completeness.

Inspect each source `go/ast.FuncType.Results` field list. Reject any result
field with a nonempty `Names` list, including `_`. This covers declarations,
function literals, nested function types, aliases, and interface methods.
Report the error at the source result name. Check source syntax, not names in
an imported `go/types.Signature`; imported Go functions remain callable and
assignable to unnamed TGo function types.

For functions with results, require explicit return operands or the existing
failure-return syntax. Keep `return` in result-free functions. Let Go check
return arity, types, and missing returns. Do not build a control-flow graph or
track assignment state for this rule. Local declarations must have initializers
regardless of later assignments.

Emit normal Go after checks. Preserve return-expression evaluation before
normal deferred calls. Deferred closures can capture explicitly initialized
locals; no new capture restriction is needed. Keep generated typed zeros for
failure returns and propagation. Do not add runtime state, constructors,
collection wrappers, or interop guards.

The [Go specification][go] defines the retained behavior. Prior decisions on
[checked fields][checked] and [failure returns][returns] provide context; their
removed implementation is not a dependency.

[WHAT](what.md) sets scope. [PROOF](proof.md) defines tests.

[go]: https://go.dev/ref/spec
[checked]: https://github.com/lucasavila00/tgo/issues/196
[returns]: https://github.com/lucasavila00/tgo/issues/81
