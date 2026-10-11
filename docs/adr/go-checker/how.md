# HOW

Use the same whole-program abstract interpreter as the
[TGo checker](../tgo-checker/how.md), with a Go policy for generated TGo types.
This gives both checkers the same call, alias, generic, and storage proofs.

Have the transpiler emit type contract metadata and source origins for every
named TGo declaration. Contracts include explicit initialization, construction
requirements, nested components, and additional type invariants. The analyzer
resolves declarations with `go/types` and attaches `go/analysis` facts to those
objects. Export and import serializable facts with declaring package paths and
type names; do not serialize object pointers. Verify imported facts against
the resolved owning package and declaration. Source aliases retain identity.
A conversion or a new defined type cannot erase a construction requirement.

Load handwritten Go and reachable dependency source with `go/packages` and
`go/types`. Generated files define the representation and facts; compiler
bookkeeping is not a handwritten Go violation. Check lowered TGo operations
with the TGo policy. Preserve origin information so generated failure slots
remain unavailable until their status proves success.

Build `go/ssa` and analyze reachable calls, allocations, stores, and loads.
Track validity by aggregate component and backing-storage interval. Preserve
aliases through interfaces, closures, and returned pointers. Follow status
relations and path conditions. Require valid state at reads, publication,
and calls that consume a protected value.

Summaries describe required input validity, writes, result validity, returned
aliases, status conditions, and generic effects. Compute recursive summaries
to a fixed point. Include every possible indirect target. Analyze actual
generic instantiations and exported generic bodies under their constraints;
unknown effects cannot prove a protected value valid.

Keep ordinary Go defaults valid unless their type contains a protected TGo
component. Propagate a dependency violation to the selected Go use, with its
origin. Inventory every analyzed operation, including explicit zeros, defaults, values
proved nonzero, and unknowns. Keep this inventory separate from policy errors.
Emit deterministic reports and record unproved boundaries.

Package facts preserve TGo rules that Go types alone cannot express. Naming
conventions or generated-file scanning cannot establish type identity and
construction validity reliably. Syntax-only checks miss helper calls and
aliases. Runtime validators add behavior and still miss unchecked paths.
A separate analysis engine would duplicate the TGo proofs; separate policies
on shared analysis give a consistent boundary instead.
