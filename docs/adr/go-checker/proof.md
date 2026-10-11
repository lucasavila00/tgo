# PROOF

Create input-driven fixtures under `tests/testdata/go-checker/`. Each fixture
contains TGo model inputs, committed generated `_tgo.go` snapshots, handwritten
Go callers and dependencies, and handwritten expected reports. Compile the TGo
inputs before checking Go; compare generated files with the snapshots.

| Area | Required cases |
| --- | --- |
| Identity | Cross-package types with equal names; aliases; embedding; new defined types |
| Construction | Valid constructor; direct literal; conversion; field mutation; constructor failure |
| Default storage | Variables; named results; omitted fields and array elements; nested aggregates |
| Allocation | New; make; full and partial fill; clear and refill; copy; reslicing |
| Presence | Map miss; closed receive; failed assertion; correct success proof |
| Calls | Safe and unsafe helpers; recursion; indirect calls; callbacks; returned closures |
| Generics | Nested protected types; wrappers; methods; constraints; inferred type arguments |
| Publication | Returned pointers; interface values; captured values; stores through aliases |

Include plain TGo structs and named scalar types, not only enums and checked
types. Reject their default Go declarations; accept complete explicit values,
including scalar zero where valid. Every rejection needs a safe companion. Repeat operations with
ordinary Go types and require ordinary zero values to pass. A generated type
must not escape checking through an alias, nested container, conversion,
interface, or generic argument.

Check cross-package fact export and import. A package must not acquire another
package's type identity because their type names match. A safe dependency use
must pass even when another use of the same function can produce a forbidden
TGo zero.

Handwrite expected positions, type identities, origins, path conditions, and
generic conditions. Check stable ordering and complete operation coverage.
Generated ABI failure slots must not create caller validity. Forwarded failed
results stay unavailable until a matching success proof.

Execute accepted fixtures against independent references to check values,
effects, panics, and status behavior. Compare bounded runtime observations with
static reports; an observed forbidden zero must never be omitted.

Record `unsafe` and cgo boundaries as outside the proof.

Run all fixture checks as project tests. Require every expected decision,
diagnostic, inventory entry, generated snapshot, and runtime comparison to pass.
