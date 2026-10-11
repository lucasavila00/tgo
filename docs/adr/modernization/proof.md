# PROOF

Add fixtures with original Go, renamed TGo, expected diagnostics,
the replacement, and a Go reference.

Cover:

- Whole-package renaming, tests, external test packages, build constraints,
  platform suffixes, imports, and Go callers.
- Direct error returns, exact wrappers, zero failure results, multiple success
  results, discarded values, and `if` initializers.
- Error reuse, shadowing, named results, bare returns, closures, addresses,
  deferred reads, labels, and backward jumps.
- Failure logging, different wrappers, nonzero failure results, and empty
  slices or maps. Do not suggest propagation.
- Conditional assignments with branch statements, initializers, nested
  branches, terminating paths, explicit types, interfaces, and named types.
- Earlier local reads or writes, captures, addresses, outer-name references,
  incomplete assignments, and jumps across the declaration.
- Disabled features and generated files. These receive no modernization
  diagnostics.

Compile and run each replacement. Compare values, ordered effects, panic,
defer order, error identity, and error text with the reference. Check source
locations and lint failure status. Recompile and lint the replacement: the
same diagnostic must not recur.

Run the upgrade sequence on a Go fixture package and its Go caller.
Require the same API and test results after conversion. Compilation
must still accept an unchanged manual form that the linter reports.
