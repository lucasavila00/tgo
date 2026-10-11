# How lowering preserves execution

The checker decides whether TGo source is valid. The lowerer decides which Go
statements implement that source. Go can accept output that calls something
too early or returns from the wrong function, so type checking alone is not
enough.

The examples below show essential behavior. Actual output uses fresh names,
explicit types, and extra storage where the surrounding code needs it.

## Call once, then check

For `use(first(), target()!, last())`, assume `target` returns `(int, error)`
and the enclosing function returns `error`:

```go
var firstValue int = first()
var value int
var err error
value, err = target()
if err != nil {
    return err
}
use(firstValue, value, last())
```

`first` runs before `target`. `target` runs once. Failure returns before `last`
or `use` runs. On success, `use` receives the saved value.

[Expression lowering](../../../internal/lowering/expressions.go) saves earlier
operands when a later operand needs statements. It preserves the order Go
specifies. It does not impose an order on operations that Go leaves unspecified.

## Check the original error type

[Propagation](../../../internal/lowering/propagation.go) checks the call's last
result before converting it to the enclosing function's return type.

For example, a nil `*E` that implements `error` is success as a pointer result.
Converting it to an `error` interface first would make the interface non-nil.
An error interface that already contains a nil pointer is non-nil and is failure.
Generic and unusual concrete error types need checks suited to their types;
reflection is used when a direct nil comparison is not available.

`!` returns the same error value. `!!` wraps it in the failure branch with
call-name context and `%w`. The private source checker verifies that the
propagated result fits the enclosing function's final error interface. Failure
returns typed zero values for results before the error. Result names and source
deferred calls retain their meaning.

## Run work only on the selected path

For `left() && right()!`, the right call and its check must remain inside the
true path of `left`. For an if value, only the selected branch runs:

```text
name := if ok { log("yes"); "yes" } else { "no" }
```

The essential Go implementation is:

```go
var chosen string
if ok {
    log("yes")
    chosen = "yes"
} else {
    chosen = "no"
}
name := chosen
```

A branch that returns or jumps out does not assign a value. The
[control-flow analysis](../../../internal/controlflow/readme.md) tells the
checker and lowerer which paths can reach a branch's last expression.

Returns, defers, and jumps inside an if value stay in the source function.
Frontend placeholders do not introduce a new function boundary. Real function
literals do have their own return targets.

## Keep loop and select behavior

[Loop lowering](../../../internal/lowering/loops.go) keeps conditions at each
loop visit. When a post statement needs several statements, generated state
skips it on the first visit and runs it before later conditions. `continue`
still reaches the post and condition.

An if value in a loop initializer can break or continue that loop. The
initializer runs on the first visit; the generated loop remains the jump target.
Variables declared with `:=` keep Go's per-iteration behavior. Renaming uses
checked bindings, so closures refer to the correct iteration variable and an
initializer can still refer to an outer name.

[Select lowering](../../../internal/lowering/select.go) evaluates channel and
send operands before selection. Receive-assignment destinations are evaluated
only in the selected case.

## Keep assignment destinations and types

[Assignment lowering](../../../internal/lowering/places.go) records destinations,
evaluates right-hand values, then performs stores. Recording a destination must
not read its value. Array assignments retain the original array; slice and map
assignments retain the evaluated container. A later store panic must not happen
before an earlier propagation can return.

Saved values use explicit checked types. For example, `var saved uint64 = 1 << n`
preserves a conversion supplied by the source context. An inferred `int` could
change the result. [Type printing](../../../internal/lowering/temporary_types.go)
uses the actual type object, because unrelated types can have the same name.
Required types must be accessible in the generated scope.

Fresh local names and import aliases avoid source declarations. Physical file
paths determine build selection; line directives affect diagnostics.

## Check behavior independently

The [test guide](../../guide/testing.md) explains the evidence:

- Snapshots show the exact generated Go.
- Handwritten outputs and Go references check values, traces, storage, errors,
  panics, jumps, and deferred calls.
- The expression proof corpus checks neutral insertion and early returns at
  recorded expression sites, including skipped paths and later loop visits.
- Rejection fixtures check invalid source and types that generated declarations
  cannot name.
- Tagged fuzz tests compare propagation and if-value programs with separate Go
  references.

Final Go checking catches invalid output. These behavior checks catch valid
output with incorrect execution. See [source checking](source-checking.md) for
checker and parser costs.
