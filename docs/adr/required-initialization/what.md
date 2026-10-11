# WHAT

Build a whole-program abstract interpreter: it tracks possible values without
running the program. Report every possible zero, including explicit `0`,
`false`, empty strings, and `nil`. Unknown values remain possibly zero.

Every variable declaration must include an initializer. Forbid named results
in TGo function declarations, literals, and types.

No operation may read or use a value supplied only by default initialization.
Explicit `0`, `false`, and nullable `nil` are initialized values.

```text
var count int          // Invalid.
var count int = 0      // Valid.
p := new(int)
*p = 0                 // Valid: assign before reading the allocation.
```

Struct literals must supply every direct field. Array and slice literals must
supply every index below their length. Apply this recursively, including aliases
and generics. Empty collections remain valid.

Allocation can reserve uninitialized storage. Accept `new(T)`, slice `make`,
and slice `clear` only when every later read follows proven initialization.
Addresses and slice headers can identify that storage; they do not initialize
its contents. Accept loop fills, initialized allocation, `append`, `copy`, and
comprehensions when their effects establish initialization. Reslicing may expose
only proven initialized elements; capacity alone is not proof.

Map reads, channel receives, and comma-ok assertions supply values only on
success. Error results also require success before use. Accept every guard and
call pattern the analysis can prove, including forwarding and delayed closure
execution. A one-result assertion supplies a value or panics.

Failure returns and propagation retain the Go ABI. Failed result slots are
unavailable storage; forwarding them preserves their failure status.

Analyze the whole program, including reachable Go dependency source, indirect
calls, and generic instantiations. Go returns are not automatically trusted.
Report dependency failures at the selected TGo package's call or use, with the
dependency origin. Reject that use if a forbidden zero remains possible; do not
require dependency changes or ban its whole package.
`unsafe` and cgo are outside the guarantee.

Zero reports and language errors are separate. Explicit zero is permitted;
using unavailable or default-filled values is an error. Static analysis can
report false positives; refine feasible proofs instead of adding source
restrictions. Keep Go types and execution order.

Write a deterministic report for every source operation with its possible-zero
components, source position, type, cause, path condition, and generic condition.
Include an inventory entry when zero is proved impossible, so omissions are
visible. Record the unsafe/cgo boundary.

[HOW](how.md) gives the analysis. [PROOF](proof.md) gives acceptance.
