# WHAT

Every operation must supply an initialized value before TGo code can use it.
Explicit `0`, `false`, and nullable `nil` are values. Default-filled storage is
not an initialized source value.

```text
var count int          // Invalid.
var count int = 0      // Valid.
p := new(int)          // Invalid.
p := new(0)            // Valid.
```

Require initializers on variable declarations. Forbid named results in TGo
function declarations, literals, and types. Functions with results must return
values; result-free functions can use bare `return`.

Struct literals must supply every direct field. Array and slice literals must
supply every index below their length. Apply this recursively, including
aliases and type parameters. Empty collections remain valid.

Reject `new(T)` and nonzero-length slice `make`; use initialized allocation,
complete literals, `append`, or comprehensions. Permit length-zero slice
allocation, map/channel allocation, and `clear(map)`. Reject `clear(slice)`;
assign explicit values instead. Reslicing can expose only elements proven
initialized, including previously initialized elements beyond the current
length. Capacity alone is not proof.

Map misses, channel receives, and comma-ok assertions require a success check
before value use, for every type. A channel range supplies only received values.
A one-result assertion either supplies a value or panics.

Results paired with `error` remain unavailable until success. Failure commas
and propagation keep the Go error ABI: generated failure slots are storage,
not usable source values. Forwarding preserves the link to the error;
a failed slot cannot become a usable value.

Generic functions must satisfy these rules for every admitted type. Type
arguments do not permit default construction.

Imported calls require an initialization contract. The contract promises
initialized successful results and describes collection writes and growth.
Go can violate that promise; there is no runtime validation. Raw imported
storage without a contract is unavailable. Imported result names do not create
TGo named results.

[HOW](how.md) gives the analysis. [PROOF](proof.md) gives acceptance.
