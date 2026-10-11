# Lower expressions into statement regions

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Context

We need a Go statement position inside expression evaluation. A flat prefix
cannot represent conditional or repeated execution.

## Decision

Build a separate Go AST emitter with these operations:

```text
emitValue(expr, region) -> value references
emitPlace(lhs, region) -> store recipe
emitStmt(stmt, region)
```

A region contains emitted statements, its source scope, and its break,
continue, and label targets. A value reference retains its Go type or untyped
constant. A store recipe retains evaluated target operands without performing
the store. Type-check the original AST once; build new output nodes.

For an expression, visit operands in Go evaluation order. Before emitting work
for a later operand, save earlier pending evaluations in fresh locals. Rebuild
the parent from those references. Keep constants in their original typing
context. Keep array locations instead of copying arrays.

Before insertion, emit `return err` at expression entry, before its operands.
After insertion, emit the expression once, discard unused results, then emit
the return. End that path; omit its remaining expression and continuation.

For `&&` and `||`, emit the left value into a local and create an `if` region
for the right operand. For example, insertion after `right()` in
`use(left() && right())` produces:

```go
r := left()
if r {
    r = right()
    return err
}
use(r)
```

Statement emitters supply the regions. Assignment prepares target operands,
evaluates RHS values, then applies store recipes in order. Delay target access
until Go performs it; preparation must not introduce an early bounds panic.

Lower affected loops into init, condition, body, and post regions. Rewrite
continues to the post entry. Lower affected expression switches into ordered
case tests and labeled bodies. Keep type switches, ranges, and selects native;
move header evaluations before them and target evaluations into their bodies.
Use native receive temporaries before evaluating selected receive targets.
Keep generated declarations in blocks that source jumps do not cross.

## Limits and validation

Select selection and communication are one native operation. This design does
not provide an insertion point between them; that boundary needs a separate
feasibility decision, not a claim of complete support.

Nested functions use separate emitters. No runtime wrappers or state machine
are needed. Compare executable traces with handwritten Go for every #248
context, then type-check and format output.
