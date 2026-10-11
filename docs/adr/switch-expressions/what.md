# Switch expressions: WHAT

Issue: [#263](https://github.com/lucasavila00/tgo/issues/263).

## Goal

Ordinary switches need repeated assignments to produce a value. Extend
[if expressions](../value-expressions.md) with switch expressions.

## Example

```text
message := switch code {
case 200:
    audit(code)
    "ok"
default:
    "error"
}
```

For `200`, audit once and supply `"ok"`. See [equivalent Go](how.md#example).

## Rules

Support initializers, expressionless switches, and type switches. Run the
initializer, then evaluate the tag once. The tag is the expression after
`switch`. Evaluate cases in Go order; select the first match. Use `default`
only when no case matches, regardless of its position.

Each completing path supplies one final value after any preceding statements.
Exiting paths need none. Preserve scopes and enclosing returns, defers,
propagation, and outer jumps. Reject `fallthrough` between value branches and
breaks targeting the value switch. Keep nested-loop breaks and ordinary
switch behavior.

Require `default` or verified enum coverage with `exhaustive:`. An enum declares
all allowed forms. Enum support is a dependency. Other switches need `default`.

Use if-expression type rules. Produce one nonconstant value. If all branches
exit, context or conversion must supply its type. General blocks remain
statements.
