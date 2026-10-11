# Expression statement-insertion spike

This package is an executable experiment for
[issue 248](https://github.com/lucasavila00/tgo/issues/248). It is separate
from the TGo compiler.

`Insert` selects an `ast.Expr` and adds ordinary Go statements immediately
before or after its run-time evaluation. The implementation lifts the selected
expression into statements in the same function. It does not use a closure.
Thus, a hook can use control flow such as `return`, and a selected `recover()`
keeps its direct caller frame.

For an after hook, the generated code first saves the expression result. The
hook runs only after normal completion and before the saved result is used. A
panic skips the hook. Earlier operands are saved before lifted statements run.
Short-circuit right operands stay conditional.

The API returns `false` without a change when the selected syntax has no
run-time evaluation. This includes constant `len` operands, `unsafe` query
operands, and an array range operand when Go does not evaluate that operand.

## Pure-Go boundary

Pure Go source cannot implement the contract for every expression. A root
deferred call is a concrete counterexample:

```go
func catcher() { observed = recover() }
func run() {
	defer catcher()
	panic(token)
}
```

Statements around the later invocation need a wrapper function. That wrapper
adds a caller frame, so `recover()` in `catcher` returns `nil`. Separate defer
statements do not solve the problem. An after defer runs even when `catcher`
panics, which violates the normal-completion rule.

A before hook on the root receive in a select case has a second source-level
boundary. A hook before the select runs for unselected cases. A hook in the
case body runs after the communication. Go has no statement position after
case selection but before the receive.

The spike reports unsupported root `go` and deferred calls. It also reports
select-entry operand cases that need a whole-select rewrite. These boundaries
are explicit. They are not compiler restrictions or fallbacks.

## Implemented evidence

The focused tests compile and run the generated Go. They cover:

- a before `return` that skips the selected expression;
- an after `return` that uses the completed selected value and skips later work;
- short-circuit evaluation;
- panic before an after hook;
- direct `recover` in a deferred function;
- contextual constants and result capture before hook mutation; and
- compile-time operands that must not get a run-time hook.

The package is a spike, not a production compiler pass. Its purpose is to show
that native statement lifting supports control-flow-changing hooks at ordinary
current-frame value evaluation points, and to record the source constructs
where pure Go has no matching statement point.
