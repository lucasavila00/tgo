# Shared typed expression lowering

Issue: [#245](https://github.com/lucasavila00/tgo/issues/245)

## Context

The emitter already has a shared expression lowerer. It returns an expression
and preceding statements, but each statement handler must decide where those
statements run. This contract does not carry enough type, scope, or execution
information. Recent loop, switch, and select fixes expose this gap.

## Decision

Keep the Go AST, Go type checker, and Go formatter. Add one typed, structured
lowering layer between type checking and Go AST emission. Do not build a new
whole-program control-flow graph or repeat Go name and type analysis.

Use these shared operations for all runtime expressions:

```text
lowerValue(expression, expected type, result count, region) -> typed values
lowerPlace(expression, region) -> typed assignment target
```

A region owns ordered generated statements at one source execution point.
It retains source scope, fresh local names, and function and branch targets.
Values and places retain types, source positions, and source object identity.
Generated nodes carry their own type facts; they must not depend on stale
type-checker maps or guesses from variable names.

The shared expression visitor lowers propagation and comprehensions once.
Statement handlers define execution regions and emit their results. They do
not implement separate expression lowering for each statement kind.

Work stays inside its region: short-circuit right operands and unmatched
switch candidates are conditional; loop tests and posts run at their required
iteration points. Keep Go iteration-variable identity. Select channel and send
operands run before selection; receive targets run only in the selected case.
Initializers and declarations keep their source visibility and lifetime.

Distinguish values from types, builtins, constants, and assignment places.
Keep contextual types, tuple results, addressable arrays, map targets, and
receiver capture. Evaluate assignment operands before stores as Go requires.
Use typed temporary variables only where evaluation order requires them.

Keep labels and jumps bound to source targets. Generated blocks must not
change scope, post execution, or valid jumps. Propagation returns from the
source function and keeps error identity. Do not hide work in function wrappers
or reject valid source to avoid lowering it.

## Evidence and alternatives

The [Go ordering pass][go-order] uses typed temporary variables and expression
initialization lists, with explicit rules for loops, select, and short circuit.
[Rust expression lowering][rust-lowering] uses destinations and continuation
blocks. Adopt its explicit destination concept, not its complete MIR machinery.
A full graph would require reconstruction of structured Go source without a
demonstrated need. A flat statement prefix does not express execution regions.

## Migration and verification

After approval, add the shared region, value, and place contracts. Move existing
expression helpers into them, then convert statement handlers. Remove each old
path when its replacement passes the same tests; do not keep a fallback.

Test nested extensions across assignments, calls, returns, conditions, loops,
switch, select, range, go, and defer. Check event order, skipped work, panic and
error behavior, contextual types, captures, scopes, and branch targets. Keep
existing regressions and compare runtime behavior where output tests cannot
prove it. Run full validation in hosted CI.

Move the approved contract to the compiler guide and specification during
implementation, and delete this ADR. This proposal does not change the language.

[go-order]: https://go.dev/src/cmd/compile/internal/walk/order.go
[rust-lowering]: https://rustc-dev-guide.rust-lang.org/mir/construction.html
