# WHAT

An `if` expression produces one value:

```text
label := if account.Active {
    log(account)
    name := account.Name
    name
} else {
    log("inactive")
    "inactive"
}
```

Branches can contain statements. The final expression supplies the value. Require `else`. Use Go headers, including initializers. Execute only the
selected branch. Ordinary `if` retains Go behavior.

`return`, `defer`, `!`, and `!!` act in the enclosing function. Labels, `break`,
and `continue` retain their targets. While computing a range collection, reject
`continue` to that range for now. Paths that cannot complete need no value
and contribute no type. Array `len`, `cap`, and `range` evaluate an `if` value.
Unsafe size and offset builtins keep their unevaluated operands.

Assignments, returns, arguments, and conversions can supply the type. Branches
must satisfy Go assignment rules. Otherwise, typed results must share
one type, and untyped constants must fit it. Use Go constant-kind rules and default types. `nil` cannot supply a type. Do not
infer interfaces or convert unrelated named types. Require explicit conversion
if the type remains unknown. If every branch terminates, require a contextual
type or conversion.

Reject multiple results and constant-only contexts, including `const` and array
lengths.

Follow [Kotlin](https://kotlinlang.org/docs/control-flow.html): conditional expressions produce values. Ordinary blocks are not expressions.
