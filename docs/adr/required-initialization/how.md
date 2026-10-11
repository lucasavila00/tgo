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

Build a control-flow graph for each source function. Track one assignment bit
per named result. Entry starts with no bits. Merge reachable predecessors by
intersection; iterate loops to a fixed point. An assignment sets a bit only
after its RHS evaluates. Compound assignments and increments first read their
target. Reads, addresses, and closure captures require the bit. Field or index
stores do not initialize an entire result. Explicit returns assign all results;
bare returns require all bits.

Include switch fallthrough, select alternatives, labeled jumps, loop post
statements, and goto edges. A loop can execute zero times. A switch without a
default and a select default have their normal paths. A terminating path does
not enter a later merge. Do not infer assignment from called functions or
closure bodies. Thus captured result storage must be initialized before a
closure is created, even when a programmer knows a later call order.

Emit normal Go after checks. Preserve named-result storage and Go defer timing.
Keep generated typed zeros for failure returns and propagation. Do not add
runtime state, constructors, collection wrappers, or interop guards.

The [Go specification][go] defines the retained behavior. Prior decisions on
[checked fields][checked] and [failure returns][returns] provide context; their
removed implementation is not a dependency.

[WHAT](what.md) sets scope. [PROOF](proof.md) defines tests.

[go]: https://go.dev/ref/spec
[checked]: https://github.com/lucasavila00/tgo/issues/196
[returns]: https://github.com/lucasavila00/tgo/issues/81
