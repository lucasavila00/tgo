# HOW

Extend `internal/frontend`. It already handles `!` and `!!` outside Go's parser.
Keep `go/parser` and `go/types`; do not copy Go's parser.

Parse `if` syntax recursively. Replace candidates with fresh marker calls
in a Go parsing copy. Parse the whole copy and header/body fragments. Use AST
context to distinguish statements from expressions. Let Go parse literal
braces. Store original syntax and positions for diagnostics.

This reuses Go's grammar. A parser copy needs upstream maintenance. Token-only
replacement cannot distinguish nested literals and branch braces reliably.

Check separate Go AST copies to find bindings and contextual types. Check
nested expressions first, then apply WHAT's branch type rule. Preserve scopes.

Inside functions, declare a fresh typed result variable. Emit an ordinary `if`
at the original evaluation point; each completing branch assigns its result.
Replace the marker with that variable. Preserve earlier operands, skipped
operands, scopes, and jump targets. Do not create a function boundary: returns,
defers, and propagation must reach the source function. Apply existing type-name
checks; report types that cannot be named.

For package initializers, call a generated function containing the branches and
source package references. Preserve Go initialization dependencies and order.

Connect through the driver. Format generated Go and type-check it.
