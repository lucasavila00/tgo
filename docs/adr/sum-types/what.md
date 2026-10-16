# Sum types: WHAT

## Goal

An enum holds one of its declared variants. Each variant can carry fields.
Extend Go's type switch to enum values.

## Example

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}

var account Account
label := switch v := account.(type) {
case Account.Personal:
    v.Name
case Account.Business:
    v.Company
exhaustive:
}
```

Here, `account` is `Personal{Name: ""}`, so `label` is `""`.
Assigning `Account.Business{Company: "Acme"}` makes the same switch produce
`"Acme"`. See [generated Go](how.md) and [tests](proof.md).

## Rules

The first declared variant is the zero value. Its fields use ordinary Go zero
values, including nil pointers. It can have any Go field type. An enum needs at
least one variant. Changing declaration order changes its zero value.

Use `default:` to handle remaining variants, or `exhaustive:` to require each
variant exactly once. [Value switches](../switch-expressions/what.md) require
one of these labels.

Evaluate the input once. Each single-variant case gives `v` that variant's type
and a copy of its fields. Changing `account` does not change this copy. Copied pointers still
point to the same data. In grouped and default cases, `v` keeps the enum type.
