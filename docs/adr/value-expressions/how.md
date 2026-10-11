# HOW

Build private `go/types` and compiled-package-reader packages from one pinned
Go version. A build script applies checked-in patches. Extend the frontend
wrapper: Go's parser reads headers and statements; marker calls retain branch
syntax and source positions.

The private checker visits branches in the source function's scope. Keep branch
constants until context resolves their types, including through operators and
shifts. The complete `if` value is nonconstant. Generic inference uses every
completing branch across conditional arguments, then checks each branch against
the parameter type. Extend scope, jump, termination, and package dependency
checks.

The driver supplies private checked packages to the lowerer. Public `Load`
retains stock `packages.Package` and checks its ordinary Go syntax separately.
Keep each check's type objects separate.

Emit typed storage and ordinary `if`:

```go
var tgo1 string
if ok {
    tgo1 = "yes"
} else {
    tgo1 = "no"
}
name := tgo1
```

Returns, defers, and propagation keep the source function. Preserve evaluation
order and package dependencies. Keep unsafe operands in constant-false blocks
to retain bindings and labels without execution; supply the builtin an expression
of the same type. Reject unnameable result types. Check emitted Go with stock Go
checking.
