# HOW

Use a whole-program abstract interpreter: compute possible values and storage
states without executing the program. Share its analysis with the
[Go checker](../go-checker/what.md); keep the TGo source rules separate.

Load selected packages and dependencies with `go/packages`. Keep source
positions and `go/types` identities. Check required initializers, result names,
and complete literals on the TGo representation before lowering. Map lowered
operations back to that representation, including generated failure slots.

Build `go/ssa`, the Go static single assignment representation. Use a
conservative call graph, refined with variable type analysis. Track possible
allocation targets for pointers, interfaces, closures, and slice backing
storage. Include every possible indirect call target. Missing bodies keep
unknown effects.

Track initialization, possible zero and nil, aggregate components, slice
intervals, and availability tied to error or presence results. A store changes
facts only after its inputs are checked. Aliases retain storage identity.
At control-flow joins, retain initialization only when every incoming path
establishes it. Preserve path conditions and refine them after tests.

Compute function summaries for input requirements, writes, returned aliases,
status-dependent results, and closure effects. Repeat recursive groups until
facts stop changing. Prove loop fills from bounds and writes, including early
exits. Widen facts to terminate recursive analysis; refine an unresolved
failure when more precise analysis can prove it safe.

Analyze reachable generic instantiations and exported generic bodies under
their constraints. Propagate effects through wrappers, callbacks, and returned
closures. Require synchronization proof before concurrent writes establish
initialization for another goroutine.

This approach can prove helper calls, loop fills, and delayed initialization.
Syntax-only bans cannot. Runtime checks change execution and do not provide a
complete static report. Separate engines for TGo and Go would duplicate alias,
call, and generic analysis; use separate policies over shared facts instead.

Write reports in stable source order. Keep dependency origins, but place errors
at selected TGo calls or uses. Never treat an unknown value as initialized.
