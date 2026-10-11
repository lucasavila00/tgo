# WHAT

Issue: [#263](https://github.com/lucasavila00/tgo/issues/263).

Extend [if expressions](../value-expressions.md) to select one value with Go
switch syntax:

```text
message := switch code {
case 200:
    audit(code)
    "ok"
default:
    "error"
}
```

Support initializers, expressionless switches, and type switches. Run the
initializer, then evaluate the tag once. The tag is the expression after
`switch`.
Evaluate cases in Go order and select the first match. Use `default` only when
no case matches, regardless of its position.

Statements can precede a branch's final expression. Each completing path
supplies one value. Exiting paths need none. Returns, defers, propagation,
and outer jumps retain their source function and targets. Reject `fallthrough`
between value branches and breaks targeting the value switch. Keep nested-loop
breaks and ordinary switch behavior.

Require `default` or verified enum coverage with `exhaustive:`. An enum declares
all allowed forms. Enum support is a dependency. Type switches and general
expression switches still require `default`.

Use if-expression rules for contextual types, constants, nil, and generic
inference. Produce one nonconstant value. If all branches exit, require context
or conversion to establish its type. General blocks remain statements.
