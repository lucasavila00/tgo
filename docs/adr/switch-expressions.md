# Switch expressions produce one value

Issue: [#263](https://github.com/lucasavila00/tgo/issues/263)

## Context

An ordinary switch needs separate assignments to supply a value. Extend
[if expressions](value-expressions.md) with switch expressions.

## Decision

Use Go headers, including initializers, expressionless switches, and type
switches. Statements can precede the final value:

```text
message := switch code {
case 200:
    audit(code)
    "ok"
default:
    "error"
}
```

Evaluate the tag once and select the first matching case in Go order.
Require `default` or verified enum coverage with `exhaustive:`. Enum support
is a dependency. Use if-expression result-type rules.

Generate fresh typed result storage and ordinary Go switches. Assign each
completing branch's value to that storage. Check case expressions only when
reached. Preserve scopes and enclosing returns, defers, propagation, and jumps.
Exiting paths need no value. Reject breaks targeting the value switch and
fallthrough between its branches. Ordinary switches retain Go behavior.

## Consequences

Generated function wrappers would change return and defer behavior. Direct
statements preserve it. General blocks remain statements.

Compile fixtures and compare results and effect traces with independent Go
references. Check tag/case order, default placement, types, scopes, outer jumps,
error identity, wrapping, and enum coverage. Require invalid-input diagnostics
and ordinary-switch regression tests.
