# Sum types: HOW

## Approach

Hide storage behind Go constructors and methods. Lower TGo construction and type
switches to these calls, so storage can change without changing callers.

## Example

```go
account := NewAccountBusiness("Acme")
switch account.Tag() {
case AccountTagBusiness:
    fmt.Println(account.BusinessPayload().Company)
}
```

Call a payload method only after selecting its variant. It returns a copy.

Give each variant a private field containing its data. Empty variants
need none. Assign tags from zero in declaration order, using the smallest
fitting unsigned integer.

Boxing stores a typed payload value in one shared `interface{}` field.
Measure the complete struct using Go's amd64 layout for every target. Include
fields, tag, interface, and padding. While larger than 80 bytes, box the largest
variant field. Choose the first variant on ties; recalculate size.
Calculate nested enums first.

Construct a fresh value containing only the selected variant. For a boxed first
variant, its method returns zero values when the tag is zero and the box is nil.

The checker compares declared variant types with case types for `exhaustive:`.
Lower matches to tag switches and payload methods.

## Tradeoffs

Unboxed payloads stay inside the enum. Boxing reduces enum copying; Go decides
whether the box needs heap allocation.
