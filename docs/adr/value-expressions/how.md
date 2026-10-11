# HOW

Pin one Go source version for `go/types` and its compiled-package reader.
A build script applies checked-in patches and generates their private packages.
Extend the existing frontend wrapper. Use Go's parser for headers and
statements. Store branch syntax and source positions beside generated call
placeholders in the syntax tree.

The private checker visits branches in the source function's scope. Retain
branch types and constants until context resolves them, including through
operators and shifts. The complete `if` value is nonconstant. Generic inference
uses every completing branch value across all conditional arguments together;
then check each branch against the inferred parameter type. Extend scope, jump, termination, and package
dependency checks.

Give the driver and lowerer private checked-package records. Public `Load` keeps
stock `packages.Package` and separately checks the same ordinary Go syntax for
lowering. These two checks keep separate type objects.

Emit a typed result variable and ordinary `if`:

```text
name := if ok { "yes" } else { "no" }
```

```go
var tgo1 string
if ok {
    tgo1 = "yes"
} else {
    tgo1 = "no"
}
name := tgo1
```

Preserve earlier and conditional evaluation. Returns, defers, and propagation
stay in the source function. Package initializer functions retain dependencies.
Reject unnameable result types. Format and check emitted Go with stock `go/types`.
