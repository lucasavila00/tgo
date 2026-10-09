# Propagate Boolean absence

Status: Proposed

## Problem

Many helpers return a value and a final `bool`. Callers often return their own zero values when the
Boolean result is false.

The syntax query helpers repeat this branch before they inspect a node:
[query.tgo](../../pkg/syntax/query.tgo#L135). The generic effect decoder repeats it inside a loop:
[generic_effects.tgo](../../internal/tgolint/generic_effects.tgo#L151). Nil-path analysis repeats it
for each path component: [nil_facts.tgo](../../internal/tgolint/nil_facts.tgo#L226).

## Decision

Extend postfix `?` to calls whose final result is `bool`:

```go
expression := ExpressionOf(node)?
kind := decodeEffectConditionKind(encodedCondition.Kind)?
base := e.nilPlace(expression.X)?
```

If the Boolean result is false, `?` returns zero values and `false` from the current function. On
success, the expression produces all call results before the Boolean result.

The current function must have a final `bool` result. This rule does not apply to a map read, a
channel receive, or a type assertion in its first version. Those forms get a second result only from
assignment context, so support for them needs a separate design.

## Constraints

- The call runs once, at its normal place in Go evaluation order.
- Lowering emits a direct Boolean check. It adds no closure, helper call, or allocation.
- The operator returns `false` unchanged. It does not convert Boolean failure to an error.
- An earlier result can have an invalid TGo zero. The existing presence rule prevents its use until
  the Boolean result is proved true.
- The same placement limits apply as for error propagation.
- `tgolint` can report the exact assignment and branch form that this operator replaces.

## Value

This change makes chains of query and validation helpers direct. It removes a temporary Boolean and
a branch without hiding which operation can fail.
