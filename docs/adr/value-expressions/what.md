# If expressions produce one value

An if value runs one branch and uses its final expression:

```text
label := if account.Active {
    log(account)
    account.Name
} else {
    "inactive"
}
```

Require `else`. Use Go headers, including initializers. Branches can contain
statements. Ordinary if statements keep their Go behavior.

## Function and jump behavior

Returns, defers, propagation, and jumps act in the enclosing source function.
A path that cannot complete supplies no value or type. A `continue` cannot
target a range while its collection is still being computed.

Array `len`, `cap`, and `range` evaluate an if value. Unsafe size and offset
builtins keep their operand evaluation rules.

## Types

An assignment, return, argument, or conversion can supply the type. Branches
must fit that type under Go assignment rules. Without such context, typed
branches must have the same type. Untyped constants follow Go's constant-kind
and default-type rules. `nil` alone supplies no type.

Do not infer a new interface or convert unrelated named types automatically.
Use an explicit type or conversion when the result type remains unknown. If
all branches terminate, a surrounding type or conversion is required.

An if value supplies one result and is not a constant. Reject multiple results
and constant-only uses, including `const` declarations and array lengths.
Ordinary blocks are not expressions.

[HOW](how.md) describes checking and lowering. [PROOF](proof.md) defines checks.
