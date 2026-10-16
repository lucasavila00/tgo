# Sum types: HOW

## Decision

Generate a Go struct with private fields and public constructors and methods:

```go
account := NewAccountBusiness("Acme")
switch account.Tag() {
case AccountTagPersonal:
    fmt.Println(account.PersonalPayload().Name)
case AccountTagBusiness:
    fmt.Println(account.BusinessPayload().Company)
}
```

Constructors take fields in declaration order and clear inactive data. Payload
methods return copies; call them after matching the tag. Tags start at zero and
use the smallest unsigned integer that fits.

The checker compares declared variants with cases. Without an explicit
`default`, the last case of a complete switch becomes Go's `default`.

Each inline variant has its own typed field. The first variant always stays
inline, so zero values and constructed values use the same representation.
Later variants start inline.

Measure the complete struct with Go's amd64 layout, including padding. Above
80 bytes, try boxing the largest later payload first. Equal payload sizes use
declaration order. Boxing stores payload values in one shared interface field.
Measure all generated fields again after each change.

Keep the smallest layout tried; equal sizes keep fewer boxes. Stop at 80 bytes
or after trying all later payloads. The target can be exceeded. Noncomparable
payloads keep the enum noncomparable. Go decides whether boxes need heap memory.
