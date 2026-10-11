# WHAT

Status: proposed. This is separate from [expression lowering][lowering].

Remove implicit source initialization, not the Go zero value. Require an
initializer for every local and package `var` declaration, including grouped
and blank declarations. Keep `:=`, parameters, receivers, range bindings,
select receives, and type-switch bindings: their operations supply values.

Choose declaration-time initialization for locals. Do not allow a local
`var x T` followed by assignments, even if all paths assign before use. This
keeps each local declaration complete and limits flow analysis to named results.

```text
var count int          // Invalid.
var count int = 0      // Valid: zero is explicit.
var owner *User = nil  // Valid for a nullable pointer.
```

Require every direct field in a struct literal, including embedded fields.
Require every index below an array or slice literal's length. Keyed literals
must have no gaps. Empty structs, zero-length arrays, empty slices, and empty
maps remain valid. Map literals need only their stated entries. Nested literals
follow these rules. Imported structs with inaccessible fields require a factory
call; a partial exported-field literal is invalid.

Keep named results, but treat them as uninitialized at function entry. Every
read and bare return must follow assignment on all incoming paths. An explicit
return assigns its results before deferred calls. A closure cannot capture an
uninitialized result; deferred assignment does not satisfy a bare return.

Keep `new(T)`, `new(expression)`, and `make`: these explicitly request Go
allocation. `make([]T, n)` explicitly requests zero-filled elements; capacity
outside the length is not readable. Missing map entries and closed-channel
receives retain Go behavior. Calls from Go can return zero values. Existing
non-nil and checked-value rules remain separate; allocation cannot bypass them.

Keep failure-return commas and `!` / `!!`: omitted non-error results become
typed zeros, including generic results. These failure values preserve the Go
error API; they do not establish a checked value's invariant.

Tradeoff: complete literals increase review detail and prevent accidental
omissions, but reduce compatibility with Go literals and field additions.
Explicit allocation retains Go APIs and generic zero construction. A ban on
all zero production would also require new collection, error, and interop
semantics. This proposal does not make that change.

Approval must settle the explicit-allocation exception and complete-literal
rule. Neither is assumed to be an accepted language rule.

[HOW](how.md) gives the implementation. [PROOF](proof.md) gives acceptance.

[lowering]: ../expression-lowering/what.md
