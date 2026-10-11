# WHAT

An `if` expression produces one value:

```text
label := if account.Active {
    account.Name
} else {
    "inactive"
}
```

Require `else`. Use Go headers, including initializers. Execute only the selected
branch. Its final expression supplies the value. Preceding statements are
allowed. Ordinary `if` statements retain Go behavior.

`return`, `defer`, `!`, and `!!` act in the enclosing function. Labels, `break`,
and `continue` retain their targets. Paths that cannot complete, such as return,
panic, or loop continue, need no value and contribute no type.

An assignment, return, call argument, or conversion can supply the type. Each
branch must satisfy Go assignment rules. Otherwise, typed results must share
one type, and untyped constants must fit it. If all results are untyped
constants, use Go constant-kind rules and default types. Bare `nil` cannot
supply a type. Do not infer interfaces or convert unrelated named types.
Require an explicit conversion if the type remains unknown. If every branch
terminates, require a contextual type or explicit conversion.

Reject multiple results and constant-only contexts, including `const` and array
lengths.

Follow [Kotlin](https://kotlinlang.org/docs/control-flow.html): conditional expressions produce values. This decision covers
`if`; switch has a separate issue. Ordinary blocks are not expressions.
