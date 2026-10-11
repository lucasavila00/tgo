# PROOF

Create input-driven fixtures under `tests/testdata/tgo-checker/`. Each directory
contains TGo inputs, any Go dependency inputs, handwritten expected decisions
and diagnostics, and expected zero inventory. Commit generated `_tgo.go`
snapshots beside their inputs; retain platform and `_test.go` filename rules.
Compare accepted programs with independent Go references.

| Area | Required cases |
| --- | --- |
| Declarations | Local, package, grouped, blank; explicit zero; missing initializer; named results |
| Aggregates | Complete and omitted nested fields and indices; aliases; empty collections |
| Allocation | Read before fill; fill before read; aliases; partial and complete loop fills |
| Slices | Clear and refill; overlapping copy; append; reslice into filled or unfilled capacity |
| Results | Map miss; closed channel; assertion; error return; forwarded status; overwritten status |
| Calls | Dependencies; recursion; callbacks; indirect targets; deferred calls; captured storage |
| Generics | Mixed constraints; nested storage; wrappers; methods; returned closures |
| Concurrency | Proven synchronization; unresolved write/read order |

Pair each forbidden case with a provably safe case. Check all operations,
including explicit zeros, operations proved nonzero, and unknown effects.
Require complete inventory and deterministic source positions. Expected reports
must be handwritten; never derive them with the checker under test.

Use bounded programs with known inputs to compare runtime zeros with the static
inventory. Every observed zero must appear in the report. This supplements the
handwritten cases; it does not prove all programs safe.

A library fixture must contain both safe and forbidden calls. Errors must point
to the selected TGo use and retain the dependency origin. Include standard
library calls whose arguments determine whether a default value is possible.

Check evaluation order, skipped work, panics, return and defer timing, `!!`
error identity, and `!` wrapping. Failed ABI result slots must stay unavailable.

Record `unsafe` and cgo boundaries without claiming they are proved safe.
Run all fixture checks as project tests. Require every expected decision,
diagnostic, inventory entry, generated snapshot, and runtime comparison to pass.
