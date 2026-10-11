# Short returns for values and errors

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

Add these return forms:

```text
return .. err       // replaces return zero, err
return value ..     // replaces return value, nil
return x, y ..      // replaces return x, y, nil
```

Success results precede the final error. Leading `..` takes one error operand
and supplies typed zero success results. Trailing `..` supplies nil. Explicit
operands must fit the corresponding result types under Go assignment rules.
A sole multiple-result expression can fill the success results.

The final result must implement `error`; trailing `..` also requires nil to fit.
Accept named interfaces, concrete errors, and constrained type parameters.
Bare `return ..` is invalid. Error-only functions can use `return .. err`.

Return unconditionally. Preserve error identity and typed nil conversions.
Evaluate operands, assign named results, then run defers. Thus this returns
`1, nil`:

```text
func Count() (count int, err error) {
    count = 9
    defer func() { count++ }()
    return .. nil
}
```

Use return syntax because the function signature supplies omitted types and
result counts without helper arguments.

[HOW](how.md) explains lowering. [PROOF](proof.md) defines checks.
