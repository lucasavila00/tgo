# HOW

Extend the frontend to parse value-producing `if`. Copy the pinned
Go parser into a private frontend package and change expression parsing to
accept this form. Keep its license and upstream version. Keep Go's scanner, token positions,
and ordinary statement grammar. Record each added expression separately and put a generated
placeholder call in the Go AST. Do not fork the Go toolchain or `go/types`.

The frontend record contains the header, branches, final expressions, source
positions, and placeholder identity. This is frontend syntax data, not a new
public API. Keep the ordinary Go AST unchanged until emission.

Use `go/types` on Go-only checking copies of the source. For each branch, replace
the placeholder with that branch's final expression and put its preceding
statements in the same lexical scope as its header and branch. These copies
are never executed. They establish bindings, operand types, and the surrounding
expected type. Check nested value expressions from the inside out. Apply the
result-type rule in WHAT, then type-check the completed Go output. Preserve
source positions so errors point to the TGo expression.

Extend the lowering operation to accept a typed branch tree. Emit the header
and branches as ordinary Go statements in the enclosing function. Each branch
that completes assigns its final value to a fresh result variable. Replace the
placeholder with that variable.

For `use(first(), if ready { left() } else { right() }, last())`, emit:

```go
firstValue := first()
var result ResultType
if ready {
    result = left()
} else {
    result = right()
}
use(firstValue, result, last())
```

`ResultType` means the checked result type. The declaration is generated Go
storage; no source path can read it before assignment.

Use the existing expression ordering pass so earlier operands run first and
skipped Boolean operands stay skipped. Emit no function wrapper: it would move
returns and defers to another function. Keep header scopes and jump targets
when generated blocks are necessary.

The driver connects frontend records, type checking, and lowering. Frontend and
lowering remain separate internal packages. Format with `go/format` and compile
with Go.

[WHAT](what.md) defines semantics. [PROOF](proof.md) defines checks.
