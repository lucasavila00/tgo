# PROOF

In `compilerv2/testdata/initialization/`, handwrite expected reports, decisions,
and positions. Give every operation zero, nonzero, and unknown cases. Run
accepted output against independent Go references.

| Operations | Required cases |
| --- | --- |
| Declarations | Local/package/grouped/blank; explicit zero/nil; missing initializer |
| Results | Reject TGo names; allow Go names; explicit and void returns |
| Literals | Complete/missing nested fields and indices; aliases; empty values |
| Allocation | Fill `new(T)` before read; reject early read; initialized allocation |
| Slice allocation | Full/partial loop fill; zero iterations; early break |
| Mutation | `clear` then refill; alias stores; ordered writes; overlapping `copy` |
| Reslicing | Shorten/extend; offsets; initialized capacity; unfilled capacity |
| Presence | Map miss, closed receive, assertions; copied and combined guards |
| Errors | Forwarded tuples; failure commas; status overwrite; escaped failed slot |
| Closures | Capture storage; initialize before invocation; invoke before fill |
| Calls | Helpers, recursion, callbacks, deferred initialization and reads |
| Dispatch | Interface/function values; all targets safe; one target unsafe |
| Generics | Nested containers, mixed constraints, inferred/explicit instantiations |
| Generic effects | Wrappers, methods, returned closures, exported entry points |
| Go dependencies | Safe/zero/unknown returns; local callsite and origin path |
| Concurrency | Channel transfers; synchronized fills; unproved write/read order |

Accept proven loop/helper/closure fills; reject unfilled and failed paths.
Explicit zeros must pass where implicit defaults fail.
Test recursive structs, nested collections, and cross-package aliases. Generic
instantiation must not hide default-filled elements.

Check report inventory completeness and deterministic ordering. Explicit
zero/false/nil reports must appear without becoming initialization errors.
Unknown targets must report possible zero. Preserve storage and status aliases.
Add cases where branch and interval refinement remove false positives.

Enumerate bounded executable programs with known inputs and call targets.
Require every observed zero to occur in the abstract report. Handwrite expected
reports independently; never generate references with the analyzer.

Test safe and forbidden calls within one dependency, plus standard-library
argument refinements. Errors must point to the TGo subpackage; no dependency
edit or package ban is required.

Compare order/count, skipped work, panics, return and defer timing,
exact `!!` identity, and single `!` wrapping. Generated failure storage must
never acquire source availability.

`unsafe` and cgo are outside these acceptance guarantees.

```sh
go -C compilerv2 test ./... -run '^TestInitialization' -count=1
```

Use hosted CI for full validation.
