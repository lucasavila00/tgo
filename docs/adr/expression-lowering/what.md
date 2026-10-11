# WHAT

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

`!!` evaluates a call, checks its error, and returns that same error before
using the call's other results. These examples run inside a function that
returns `error`.

## Inside a call

Source:

```text
join(first(), target()!!, last())
```

Generated Go:

```go
firstValue := first()
targetValue, err := target()
if err != nil {
    return err
}
join(firstValue, targetValue, last())
```

`first()` runs before `target()`. If `target()` fails, neither `last()` nor
`join()` runs.

## Inside a conditional expression

`left()` returns `bool`. `right()` returns `(bool, error)`.

Source:

```text
use(left() && right()!!)
```

Generated Go:

```go
r := left()
if r {
    rightValue, err := right()
    if err != nil {
        return err
    }
    r = rightValue
}
use(r)
```

If `left()` is false, `right()` does not run. If `right()` fails, `use()` does
not run. Otherwise, `use()` receives the Boolean result.

#248 proves after-evaluation statement insertion. #247 adds the error check
and `!` / `!!` syntax. No before-insertion API is needed.

[HOW](how.md) chooses the tools. [PROOF](proof.md) defines the tests.
