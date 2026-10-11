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
The result need not be a compile-time constant, even when its branches contain
constants. If no branch can complete, use the surrounding expected type;
without one, require an explicit conversion.

This decision restores the `if` part of the language proposal from
[commit 4578415](https://github.com/lucasavila00/tgo/blob/4578415333b40c0395b99ae1e7399f601cda06e6/docs/adr/value-producing-control-flow.md)
and makes its type rule explicit.

[HOW](how.md) defines implementation. [PROOF](proof.md) defines checks.
