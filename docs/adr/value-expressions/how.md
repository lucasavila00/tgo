# HOW

Extend the frontend wrapper for `!` and `!!`. Recognize expression `if`.
Use `go/parser` for headers and ordinary branch statements. Handle nested
expressions recursively. Record source positions. Put a generated call in the
surrounding Go syntax tree to stand for each expression until its type is known.

Check branch values in their original assignment, call, return, or conversion
context with `go/types`. Keep source bindings and apply WHAT's type rule.
The Go representation used to check these branch values is still undecided.

Replace the expression with a fresh typed result variable and an ordinary `if`.
For:

```text
name := if ok { "yes" } else { "no" }
```

emit:

```go
var tgo1 string
if ok {
    tgo1 = "yes"
} else {
    tgo1 = "no"
}
name := tgo1
```

Each completing branch assigns its final value. Keep earlier operands first
and conditional operands conditional. Preserve scopes and jump targets.
Returns, defers, and propagation stay in the source function.
Report result types that cannot be named.

For package initializers, use generated functions that retain source package
references, initialization dependencies, and order.

Connect through the driver. Format generated Go and type-check it.
