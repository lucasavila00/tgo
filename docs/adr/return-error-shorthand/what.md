# Short returns: WHAT

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

## Goal

Omit repeated zero success values and nil errors.

## Example

```text
return .. err       // replaces return zero, err
return value ..     // replaces return value, nil
return x, y ..      // replaces return x, y, nil
```

## Rules

Success results precede the final error. Leading `..` takes one error
operand and supplies typed zero success results. Trailing `..` supplies nil.
Explicit operands must fit corresponding result types under Go assignment
rules. A sole multiple-result expression can fill the success results.

The final result must implement `error`; trailing `..` also requires nil to
fit. Accept named interfaces, concrete errors, and constrained type
parameters. Bare `return ..` is invalid. Error-only functions can use
`return .. err`.

Both forms require unnamed results. Reject either form if any result
parameter has a name, including `_` and grouped names.

Return unconditionally. Preserve error identity, typed nil conversions, and
deferred calls. The function signature supplies omitted types and result
counts.

[HOW](how.md) explains lowering. [PROOF](proof.md) defines checks.
