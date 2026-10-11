# WHAT

Status: proposed; separate from [expression lowering][lowering].

Remove implicit source initialization, not the Go zero value. Require an
initializer for every local and package `var` declaration, including grouped
and blank declarations. Keep `:=`, parameters, receivers, range bindings,
select receives, and type-switch bindings: their operations supply values.

Require declaration-time initialization for locals. Reject `var x T` even if
later assignments cover all paths. No definite-assignment analysis is needed.

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

Forbid named results in all TGo-authored function declarations, function
literals, and function type signatures, including interface methods. Reject
blank result names too. Returns from functions with results must supply their
values; a result-free function can use `return`. Keep failure-return commas as
explicit failure syntax. Imported Go signatures can have result names: these
are not TGo source declarations and do not restrict calls or assignments.

Deferred closures can read or change explicitly initialized locals. Return
expressions evaluate before deferred calls; a later local assignment does not
replace an already returned value. Reference values retain normal Go behavior.
Consider named results later together with deferred local initialization.

Keep `new(T)`, `new(expression)`, and `make`: these explicitly request Go
allocation. `make([]T, n)` explicitly requests zero-filled elements; capacity
outside the length is not readable. Missing map entries and closed-channel
receives retain Go behavior. Calls from Go can return zero values. Existing
non-nil and checked-value rules remain separate; allocation cannot bypass them.

Keep failure-return commas and `!` / `!!`: omitted non-error results become
typed zeros, including generic results. These failure values preserve the Go
error API; they do not establish a checked value's invariant.

Tradeoff: complete literals prevent omissions but reduce Go compatibility.
Explicit allocation retains zero production. A complete zero-value ban would
also change collection, error, and interop semantics.

Approval must settle the explicit-allocation exception and complete-literal
rule. Both require approval.

[HOW](how.md) gives the implementation. [PROOF](proof.md) gives acceptance.

[lowering]: ../expression-lowering/what.md
