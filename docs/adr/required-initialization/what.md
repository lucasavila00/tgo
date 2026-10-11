# WHAT

Every variable declaration must include an initializer.

```text
var count int          // Invalid.
var count int = 0      // Valid.
var owner *User = nil  // Valid for a nullable pointer.
```

Later assignments do not satisfy this rule, even if every path assigns before
use. Do not add definite-assignment analysis.

Forbid named results, including `_`, in TGo function declarations, function
literals, and function type signatures. This includes interface methods.
Functions with results must return explicit values. Result-free functions can
use bare `return`. Imported Go result names do not restrict calls.

Struct literals must supply every direct field, including embedded fields.
Array and slice literals must supply every index below their length. Keyed
literals cannot have gaps. Apply these rules to nested literals. Empty values
remain valid when they have no fields or elements. Maps require only their
stated entries. Imported structs with inaccessible fields need factory calls.

Keep explicit `new`/`make` allocation, including zero-filled slices, and typed
zeros in failure returns and error propagation. Keep Go interop, map misses, and closed-channel behavior.

Deferred closures can change initialized locals. Return expressions evaluate
before defers. Returned references retain normal Go behavior.

[HOW](how.md) gives the implementation. [PROOF](proof.md) gives acceptance.
