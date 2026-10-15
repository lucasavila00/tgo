# Sum types: WHAT

## Goal

Store payloads inline and match every form.

## Example

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}
var account Account = nil
label := switch v := account {
case Account.Personal: v.Name
case Account.Business: v.Company
case nil: "none"
exhaustive:
}
```

This produces `"none"`. `Account.Personal{Name: "Lucas"}` produces `"Lucas"`.
See [generated storage](how.md) and [checks](proof.md).

## Rules

Every enum includes a payload-free, typed `nil`. `account == nil` tests tag zero;
assign nil contextually. Enum zero is permitted:
`var`, `new`, arrays, `make`, map misses, closed channels, generic `var T`,
and error-return zeros all produce nil.

`exhaustive:` requires each declared variant and nil exactly once. Otherwise,
[switch expressions](../switch-expressions/what.md) need `default`.

Evaluate the input once. Each single-variant clause binds `v` to a payload copy.
Grouped, default, and nil clauses have no `v`. Only that copy permits payload
access. Mutating the source cannot invalidate it. Copies retain Go reference sharing.

Payload construction still requires non-null and checked values. Nil proves
no payload contract; Go callers must preserve valid tags and payload contracts.
Unknown tags are invalid; they do not mean nil. Other initialization
rules remain separate.
