# How lowering preserves execution

Lowering replaces `call()!!` and `call()!` with ordinary Go statements: call
once, test its error, return on failure, then use its success results. The rules
below keep these statements at the point where the source call runs.

## Calls and propagation

For `use(first(), target()!!, last())`, assume each success result is `int` and
the call's final result is `error`. The essential generated code is:

```go
var tgo1 int = first()
var tgo2 int
var tgo3 error
tgo2, tgo3 = target()
if tgo3 != nil {
    return tgo3
}
use(tgo1, tgo2, last())
```

`first()` runs before `target()`. The tuple assignment calls `target()` once.
If it fails, neither `last()` nor `use()` runs. Otherwise, the parent receives
the saved success result. [Operand lowering](../../../internal/lowering/expressions.go)
saves earlier operands when a later operand needs statements. This preserves
Go's specified call order; it does not promise an order that Go leaves unspecified.

[Propagation](../../../internal/lowering/propagation.go) checks the call's final
result before conversion to the caller's return type. A nil `*E` would become a
non-nil `error` interface after conversion, so checking the converted value
would give the wrong answer. An interface that already contains a nil pointer
is failure. Generic results follow their instantiated types; reflection is
used when a direct nil comparison is not available. `!!` returns the original
error. `!` wraps it once, inside the failure branch.

The first type check uses helpers that expose only success results. It cannot
check a return statement that lowering has not inserted yet. The compiler
therefore also uses Go's assignability rule to check the propagated error
against the caller's final result. Wrapped propagation checks the wrapped
`error` type.

## Conditional execution and function boundaries

The expression transform keeps a short-circuit right operand inside its branch.
For `left() && right()!!`, `right()` and its error check run only if `left()` is
true. A single unconditional prefix would be wrong.

[Statement lowering](../../../internal/lowering/statements.go) places repeated loop
conditions inside the loop. When the post statement needs several statements,
a generated Boolean skips the post on the first visit and runs it before each
later condition. Thus `continue` still leads through the post and condition.
[Select lowering](../../../internal/lowering/select.go) evaluates send and receive
operands before selection, but performs receive-target work only in the chosen case.

Named functions and function expressions each get their own return target.
An inner error returns from the inner function. Generated returns remain
ordinary Go returns, so source deferred calls still run. Package initializers
keep their function expressions and surrounding evaluation order.

## Assignment, types, and scope

[Storage lowering](../../../internal/lowering/places.go) captures left-hand operands,
then evaluates right-hand values, then performs stores. Capturing a destination
must not read it. An array store keeps the original array; a slice or map store
keeps the evaluated container value. Explicit address evaluation completes its
checks before an inserted statement. Store checks remain at the store, so an
early return does not cause a later bounds or map-key panic.

Every saved expression uses an explicit checked type. This preserves contextual
shifts and named types. [Type printing](../../../internal/lowering/temporary_types.go)
uses type identity: two types called `T` are not necessarily the same type.
Fresh names avoid source bindings. Return variables use function-entry scope,
where parameters and result names are visible. Labelled declarations keep their
original scope.

[File handling](../../../internal/driver) uses physical paths and positions for
selection. Line directives change diagnostic locations, not source identity.
The pass checks that every accepted marker was processed; a skipped function
cannot silently retain a helper that discards its error.

## Checks of these rules

[The proof matrix](../../../tests/proof_test.go) inserts neutral statements and early
returns at expressions in the committed inventory. Handwritten traces check
unchanged effects and the exact first or later visit that exits. Every generated
variant must compile. Storage, panic, defer, and jump cases check where simple
Go type checking cannot detect a change in behavior.

[Propagation tests](../../../tests/propagation_test.go) check snapshots and handwritten
output. [Separate Go references](../../../tests/temporary_types_test.go) check the
corresponding fixtures. Rejection tests cover incompatible returns, inaccessible
temporary types, and propagation in unsafe size or offset operands.

[The fuzzer](../../../tests/lowering_fuzz_test.go) compares bounded programs with
independent references. It excludes source `recover`, goroutines, and nested propagating functions.
