# Sum types: WHAT

## Goal

Declare enums at package level. An enum holds one declared variant and its
fields. Use Go's type
switch syntax to select a variant and read its fields.

## Example

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string }
}

account := Account.Business{Company: "Acme"}
label := switch v := account.(type) {
case Account.Personal:
    v.Name
case Account.Business:
    v.Company
exhaustive:
}
```

`label` is `"Acme"`. A zero `Account` is `Personal{Name: ""}`.

## Rules

The first variant is the zero value. Its fields have Go zero values. An enum
needs at least one variant. Changing variant order changes the zero value.

Use `default:` for remaining variants, or an empty `exhaustive:` clause to
require every variant exactly once. Value switches require one of these clauses.

The switch reads its input once. A single-variant case gives `v` a copy of that
variant's fields. Replacing `account` does not change this copy. Copied pointers
still refer to the same data. In grouped and default cases, `v` keeps the enum
type.
