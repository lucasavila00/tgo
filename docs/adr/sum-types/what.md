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

This produces `""`: zero selects `Personal`, with an empty `Name`.
`Account.Personal{Name: "Lucas"}` produces `"Lucas"`.
See [generated storage](how.md) and [checks](proof.md).

## Rules

Enums declare at least one variant. The first has tag zero. Its
payload fields have their ordinary Go zero values: numbers are zero, strings
are empty, and pointers are nil. The payload can contain any Go field type.
Declaration order determines the zero value.

`var`, `new`, arrays, `make`, map misses, closed channels, generic `var T`, and
error-return zeros all produce this first variant with zero fields.

Use `default:` to handle remaining variants, or `exhaustive:` to require a case
for every declared variant. [Switch expressions](../switch-expressions/what.md)
require one of these labels.

Evaluate the input once. Each single-variant clause binds `v` to a payload copy.
Grouped and default clauses have no `v`. Only that copy permits payload
access. Mutating the source cannot invalidate it. Copies retain Go reference sharing.
