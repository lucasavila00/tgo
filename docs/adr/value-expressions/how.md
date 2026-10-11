# Check branch values, then emit statements

The frontend prepares if syntax for Go's standard parser. Markers carry branch
statements and source positions to the private checker. Their placeholder
functions do not create source return, defer, or jump boundaries.

The checker visits branches in the source function's scope. It keeps constants
until the surrounding type is known and uses completing branches for generic
inference. It checks each branch against the resolved type, along with scope,
jumps, and paths that terminate.

The driver supplies checked source to the lowerer. For a simple value:

```text
name := if ok { "yes" } else { "no" }
```

emit the behavior of:

```go
var chosen string
if ok {
    chosen = "yes"
} else {
    chosen = "no"
}
name := chosen
```

Returns, defers, propagation, and jumps keep their enclosing source function.
Preserve order relative to surrounding operands and package initializers.
Required result types must be nameable in generated Go.

For an unevaluated unsafe operand, retain bindings and labels in a constant-false
block and give the builtin an expression of the same type. Do not execute branch
work that the source operand does not execute.

The driver writes output and checks it with stock Go. See
[source checking](../../arch/transpiler/source-checking.md) and
[lowering](../../arch/transpiler/correctness.md) for the implementation.

[WHAT](what.md) states the language rules. [PROOF](proof.md) defines checks.
