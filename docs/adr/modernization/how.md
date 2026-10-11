# HOW

Add `tgolint` for handwritten TGo source. Use the frontend's source positions
and the driver's checked types and bindings. Exclude generated Go. Report
the rule, source location, and replacement. A modernization diagnostic makes
the lint command fail; it does not make compilation fail.

Start with a call assignment followed by `if err != nil { return ... }`,
including an `if` initializer. Require zero failure results and no additional
failure effects. Check every use of the error binding, named results, closures,
addresses, deferred reads, and reachable bare returns. Preserve success
bindings and their scope. Choose `!!` for error identity; choose `!` only for
the exact compiler wrapper.

For `if`, require an uninitialized local declaration followed by a complete
conditional whose completing branches end in assignment to that local.
Keep the declared type through an explicit typed initializer. Branch statements
stay in order. Check binding identity, earlier reads and writes, captures,
addresses, labels, and jumps before removing the original declaration.
Do not change what an outer name denotes.

Reuse the v0 rules as design references, not as a compiler dependency:
[error propagation](https://github.com/lucasavila00/tgo/blob/4858dde42883c0acd76f02340f304813814472ce/internal/tgolint/modernize_error_return.tgo).

Report only replacements proven equivalent. Unknown cases retain their source.
