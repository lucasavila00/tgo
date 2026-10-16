# Sum types: WHAT

## Goal

Store payloads inline and match every form.

## Example

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}
var account Account
label := switch v := account {
case Account.Personal: v.Name
case Account.Business: v.Company
exhaustive:
}
```

This produces `""`: zero selects `Personal`, whose `Name` is an empty string.
`Account.Personal{Name: "Lucas"}` produces `"Lucas"`.
See [generated storage](how.md) and [checks](proof.md).

## Rules

An enum declares at least one variant. The first variant has tag zero. Its
payload fields have their ordinary Go zero values: numbers are zero, strings
are empty, and pointers are nil. The payload can contain any Go field type.
Declaration order therefore determines the enum's zero value.

`var`, `new`, arrays, `make`, map misses, closed channels, generic `var T`, and
error-return zeros all produce this first variant with zero fields.

`exhaustive:` requires each declared variant exactly once. Otherwise,
[switch expressions](../switch-expressions/what.md) need `default`.

Evaluate the input once. Each single-variant clause binds `v` to a payload copy.
Grouped and default clauses have no `v`. Only that copy permits payload
access. Mutating the source cannot invalidate it. Copies retain Go reference sharing.
