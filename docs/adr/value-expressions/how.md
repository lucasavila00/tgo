# HOW

Keep standard `go/parser` and `go/types`. Add a TGo extension parser in
`internal/frontend`; do not copy Go's parser. Scan tokens, parse the added `if`
forms recursively, and keep their headers, branches, final expressions, and
original source positions in frontend records.

Replace each candidate with a fresh marker call in a Go-only parsing copy.
Parse the complete copy and each header/body fragment with `go/parser`. Use
those AST contexts to distinguish statement `if` from expression `if`. Preserve
ordinary statements. Let Go parsing distinguish composite literals and function
literals from branch braces; brace matching alone is insufficient. Map fragment
positions and diagnostics back to the original source.

This approach reuses Go's maintained grammar and limits our parser to the added
syntax. A private Go parser copy would provide one grammar and direct positions,
but requires continuous upstream updates. A token-only rewrite cannot reliably
classify nested headers and blocks. Use neither alternative.

Use Go-only checking copies to establish branch bindings, operand types, and
contextual result types. Keep header and branch locals in their original scopes.
Check nested expressions inside out, apply WHAT's type rule, then check the
completed output with `go/types`. Checking copies are never executed.

In a function, emit ordinary `if` statements at the expression's evaluation
point. Each completing branch stores its final expression in a fresh result
variable. Replace the marker with that variable. Use the existing typed-storage
lowering for private and generic types; do not require their source spelling.
Earlier operands must execute first. Short-circuit operands must stay skipped.
Do not introduce a function boundary: returns, defers, and propagation belong
to the source function. Preserve scopes and jump targets.

For a package initializer, emit a generated initializer function with the same
branch logic and call it from the original declaration. Keep all source package
references visible in its body so Go retains initialization dependencies. Keep
declaration order and evaluate branch effects only when selected. There is no
surrounding function return or defer scope to cross.

For `label := if active { "on" } else { "off" }`, generated Go uses a fresh name:

```go
var tgoResult string
if active {
    tgoResult = "on"
} else {
    tgoResult = "off"
}
label := tgoResult
```

The driver connects frontend records, type checking, and lowering. Frontend and
lowering remain separate internal packages. Format with `go/format`, then compile
with Go.

[WHAT](what.md) defines semantics. [PROOF](proof.md) defines checks.
