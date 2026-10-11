# HOW

Load the build target and reachable dependencies with [`go/packages`][packages],
including syntax and `go/types.Info`. Preserve source object identities and
positions. Check TGo declaration initializers, result names, and literal
completeness before lowering. Dependency Go source has no TGo spelling rules,
but its default-filled values remain uninitialized in the analysis.

Build [`go/ssa`][ssa] for the lowered program and dependency bodies. Map generated
operations back to source; mark failure-return storage explicitly. Use
[`vta`][vta] to refine a conservative call graph. Add allocation-site points-to
sets for pointers, interfaces, closures, and slice backing storage. Include all
possible indirect targets; unavailable ordinary bodies retain unknown effects.

For each value, track zero/nonzero possibilities, nil/non-nil possibilities,
initialization, and conditional availability. Unknown includes every possibility.
Track aggregate components, intervals, and success relations.
A store establishes facts after
its RHS and destination checks. Loads require facts. Header copies retain
backing-storage identity; shortening retains initialized intervals. `clear`
removes initialization facts for affected elements. `copy` transfers only the
source's initialized interval. Follow aliases and recursive type dependencies.

Interpret SSA instructions over those facts. Union possible values at joins;
intersect guaranteed initialization. Retain path predicates for refinement.
Solve loop induction
variables and bounds to prove complete fills, including early exits. Translate
slice-relative bounds to backing-storage offsets. Accept reslicing when the
exposed interval is initialized. Widen recursive intervals to ensure termination;
refine the failing path when a coarse result loses a usable proof.

Compute function summaries for required initialized inputs, writes, returned
aliases, conditional results, and closure effects. Iterate recursive call groups
to a fixed point; specialize summaries by argument facts where needed. Apply
summaries across Go and TGo calls, defers, callbacks, and channel transfers.
Concurrent writes establish facts only with proved synchronization.

Instantiate reachable generics and retain constraint-safe summaries for exported
generic bodies. Check every admitted type case, including nested containers;
inferred and explicit type arguments use the same analysis. Analyze exported
entry points with required input facts.

Propagate dependency zero origins through summaries. Diagnose the selected TGo
call/use, with its dependency path and instantiation; refine safe arguments and
status paths instead of reporting errors in dependency files.

Emit `zero-report.json` sorted by source file and position. Inventory each
operation, including generated ABI storage separately; record causes, types,
paths, generic conditions, and boundary assumptions. Unknown is never silently
nonzero or initialized. Refine feasible proofs without losing possibilities.
No runtime checks.

[packages]: https://pkg.go.dev/golang.org/x/tools/go/packages
[ssa]: https://pkg.go.dev/golang.org/x/tools/go/ssa
[vta]: https://pkg.go.dev/golang.org/x/tools/go/callgraph/vta
