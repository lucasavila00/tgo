# Switch expressions: HOW

## Approach

Generate fresh typed result storage and ordinary Go switches. Replace completing
final expressions with result assignments.

Extend conditional records with expression switches, type switches, and source
positions. Go parsing reads headers and clauses. Parsing placeholders must not
create source function boundaries.

The private checker checks branches in their original scopes. Initializer
bindings cover the switch; type-switch bindings retain Go's clause-specific
types. Record result types, completing paths, and jump targets. Verify enum
coverage against resolved declarations.

Reuse [expression lowering](../expression-lowering.md) to preserve earlier
operands, skipped paths, assignments, and repeated loop evaluation. Package
initializers use existing initializer rules.

Retain the Go switch when cases need no added statements. Otherwise, save its
tag once. Check cases in order while none has matched. Save the selected clause
number; dispatch with an ordinary switch. Select default after all cases fail.
Retain Go type switches.

## Example

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

## Tradeoffs

A generated function wrapper would change source returns, defers, and jumps.
Direct statements preserve their targets.
