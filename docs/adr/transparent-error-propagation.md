# Pass errors without a wrapper

Status: Proposed

## Problem

Postfix `!` removes a manual error branch, but it wraps the error with the call name. Some code must
return the same error without a new wrapper.

The comprehension parser has three direct error branches in one function:
[parser_comprehensions.tgo](../../pkg/syntax/parser_comprehensions.tgo#L158). The syntax builder has the
same form after calls to `parseExpression`:
[parser_build.tgo](../../pkg/syntax/parser_build.tgo#L82). These branches cannot use `!` because that
change would add error context.

## Decision

Add postfix `?` for a call whose final result has the predeclared `error` type:

```go
bodyOpen := p.openToken(bodyClose)?
clause := p.rawComprehensionClause(start, bodyOpen, bodyClose)?
children, result := p.rawComprehensionBody(bodyOpen+1, bodyClose)?
```

If the error is not nil, `?` returns the original error. It supplies the zero value for each earlier
result of the current function. On success, the expression produces all call results before the
error.

The current function must have a final `error` result. The call result and function result rules are
otherwise the same as the rules for `!`.

## Constraints

- The call runs once, at its normal place in Go evaluation order.
- `?` does not wrap, copy, or replace the error.
- Lowering emits a direct error check. It adds no closure, helper call, or allocation.
- `?` has the same placement limits as `!` for `go`, `defer`, `select`, switch cases, and loop control.
- `!` keeps its current meaning. Use `!` when the call name is useful error context.

## Value

This change removes repeated branches from parser code and keeps the required error identity. The
source also shows the choice at the call site: `!` adds context, and `?` passes the error unchanged.
