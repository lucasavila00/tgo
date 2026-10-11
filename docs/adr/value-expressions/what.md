# WHAT

Allow `if` to produce one value in expression positions. The ordinary statement
form keeps Go behavior. A branch's final expression supplies its value.

```text
label := if account.Active {
    account.Name
} else {
    "inactive"
}
```

Use Go headers, including initializer statements. Require `else`. Evaluate
only the selected branch. Every path that completes the expression must supply
one value. A path that returns, panics, or continues an enclosing loop does not
need a value.

A branch can contain ordinary statements before its final expression. `return`,
`defer`, `!`, and `!!` operate in the enclosing function. Labels, `break`, and
`continue` keep their targets.

A surrounding assignment, return, call argument, or conversion can supply the
result type. Check each branch against that type with Go assignment rules.
Without such a type, all typed branch results must have the same type; untyped
constants must fit that type. If all results are untyped constants, apply Go's
constant-kind rules and default type. Bare `nil` cannot determine a type.
Require an explicit conversion when no single type can be determined. Do not
infer a new interface or convert between unrelated named types. Branches that
cannot complete do not contribute a result type.

The expression produces one result. An ordinary block is not an expression.
An `if` expression is not a Go constant, even when its branches contain
constants. Reject it where Go requires a constant, including `const` declarations
and array lengths. If no branch can complete, use the surrounding expected type;
without one, require an explicit conversion.

The design follows [Kotlin conditional expressions](https://kotlinlang.org/docs/control-flow.html),
where `if` and `when` produce values. This ADR covers `if`; the second phase
covers the switch-like construct. Only these conditional constructs produce
branch values. Do not add general block expressions.

[HOW](how.md) defines implementation. [PROOF](proof.md) defines checks.
