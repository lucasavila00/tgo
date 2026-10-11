# HOW

Generate typed result storage and ordinary Go switches:

```go
var result string
switch code {
case 200:
    audit(code)
    result = "ok"
default:
    result = "error"
}
message := result
```

Use fresh names. A generated function wrapper would change source returns,
defers, and jumps. Direct statements preserve their targets. Package
initializers use the existing initializer rules.

Extend conditional records with expression switches, type switches, and source
positions. Go parsing still reads headers and clauses. Parsing placeholders
must not create source function boundaries.

The private checker checks branches in their original scopes. Initializer
bindings cover the switch; type-switch bindings retain Go's clause-specific
types. Record result types, completing paths, and jump targets. Verify enum
coverage against resolved declarations, not member spelling.

Replace completing final expressions with result assignments. Reuse
[expression lowering](../expression-lowering.md) to preserve earlier operands,
skipped paths, assignments, and repeated loop evaluation.

Retain the Go switch when cases need no added statements. Otherwise, save its
tag once. Check cases in order while none has matched, then save the selected
clause number. Dispatch with an ordinary switch. Select default only after all
cases fail. Keep type switches as Go type switches.
