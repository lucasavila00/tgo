# WHAT

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

Evaluate the selected expression once, then insert `return err`.

## Inside a call

Insert after `target()` in `join(first(), target(), last())`:

```go
firstValue := first()
targetValue := target()
return err
join(firstValue, targetValue, last())
```

`first()` and `target()` run. `last()` and `join()` do not. The unreachable
last line keeps the temporary variables used.

## Inside a conditional expression

Insert after `right()` in `use(left() && right())`:

```go
r := left()
if r {
    r = right()
    return err
}
use(r)
```

If `left()` is false, skip `right()` and the return. Otherwise, run `right()`
and return.

Only after-evaluation insertion is needed. #247 adds the error check around
`return err` and handles `!` / `!!` syntax.

[HOW](how.md) chooses the tools. [PROOF](proof.md) defines the tests.
