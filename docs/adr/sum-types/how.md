# Sum types: HOW

## Decision

Generate a Go type with private storage, constructors, and methods. This lets
Go callers use enums without depending on their fields.

For the [Account example](what.md), Go callers write:

```go
account := NewAccountBusiness("Acme")
switch account.Tag() {
case AccountTagPersonal:
    fmt.Println(account.PersonalPayload().Name)
case AccountTagBusiness:
    fmt.Println(account.BusinessPayload().Company)
}
```

Payload methods return copies. Call them after selecting the matching tag.
Constructors take fields in declaration order and clear other variants' data.
Tags start at zero and use the smallest unsigned integer that fits.

The checker identifies each variant by its declaration. It checks construction,
case fields, and complete coverage before generation. A complete enum switch
becomes a Go tag switch. Without an explicit `default`, its final case becomes
`default`, so Go also recognizes when every branch returns.

## Open storage decision

Current storage keeps all payloads inside the enum. Automatic boxing would move
large payloads into an interface field. Decide whether to keep the first payload
inside the enum and treat 80 bytes as a target. Set the equality and map-key
rules before enabling boxing: zero values and constructed values must agree,
without losing floating-point negative zero.
