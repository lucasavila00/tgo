# Add statements at an expression's evaluation point

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

Error propagation needs statements inside larger expressions. The lowerer must
run those statements where the original expression executes, without repeating
it or moving later work ahead of an error return.

## Inside a call

For a function that returns `error`:

```text
join(first(), target()!, last())
```

The generated Go must have this behavior:

```go
firstValue := first()
targetValue, err := target()
if err != nil {
    return err
}
join(firstValue, targetValue, last())
```

`first` runs before `target`. Failure returns before `last` or `join` runs.
Actual output uses fresh names and checked types.

## Inside a skipped operand

If `left` returns `bool` and `right` returns `(bool, error)`:

```text
use(left() && right()!)
```

The behavior is:

```go
value := left()
if value {
    rightValue, err := right()
    if err != nil {
        return err
    }
    value = rightValue
}
use(value)
```

`right` runs only when `left` is true. Putting its check before the condition
would change execution.

The same placement rule supports the public `InsertAfter` API and TGo lowering.
[HOW](how.md) describes the implementation. [PROOF](proof.md) defines checks.
