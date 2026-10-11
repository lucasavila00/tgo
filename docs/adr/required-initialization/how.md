# HOW

Analyze source with `go/ast`, `go/types.Info`, and source object identity before
lowering. Inspect declarations, literals, builtin calls, reads, assignments,
conversions, and function signatures. Keep generated ABI storage outside source
initialization checks.

Represent facts for initialized values, pending result pairs, and initialized
collection intervals. Struct and array facts include their fields/elements;
resolve aliases and recursively follow underlying types. Pointer facts refer
to allocation objects. Slice facts share backing-storage objects and track
bounds, length, and capacity. Copying a header does not copy storage facts.

Build function control-flow graphs with edges for loops, switches, selects,
labels, and jumps. Merge facts true on every incoming path; iterate loops to
stability. Transfer facts after RHS evaluation and ordered stores. Invalidate
facts affected by an unknown write or escape. Keep initialized intervals after
slice shortening. Translate slice-relative indices to backing-storage offsets.
Recognize constant ranges, same-slice `len` bounds, and guarded symbolic bounds.
An existing initialized interval can justify reslicing beyond current length;
otherwise require assignments that establish the interval.

Track the relation between each conditional value and its success status.
Conditions establish availability; stopping failure branches permit later use.
Follow copies, aliases, captured objects, and forwarded results without requiring
a particular guard spelling. Analyze closures at their execution point or
export their preconditions. Keep a value unavailable if the relation is lost.
Generate failure storage only at ABI returns; it never establishes source
availability. Preserve defer timing and propagation error identity.

Analyze generic bodies over normalized constraint type sets. Record summary
effects for zero construction, initialization, mutation, returned aliases, and
pending pairs. Instantiate effects at both inferred and explicit calls; carry
them through wrappers, methods, and returned function values. Mixed constraints
must satisfy every admitted case; an unknown type never implies a valid zero.

Infer imported summaries from available Go source using the same analysis.
Export summaries with package facts. For unavailable or unprovable foreign
bodies, require a supplied contract with the same facts; reject calls and
storage reads without one. Check function values and interface dispatch against
all possible targets or their declared contract. Contracts are the explicit
foreign trust boundary, not a proof of foreign execution.

Use one analysis for direct and generic operations. Report the operation that
lacks an initialization fact; do not add runtime guards or change Go types.
