# tgo

Archived proposal. See the [current language specification](../../spec/README.md).

A TypeScript-like tool for Go. Declare business types and rules in less code.
Compile a small business-logic package to Go; keep the rest of the application in Go.

Not a Go source superset. tgo changes initialization rules and adds sum types.
Its focus is easier review: valid states, construction rules, and business decisions.

## 1. Sum types

Each variant has its own fields. Require every match case.
Variants can contain Go structs, collections, pointers, interfaces, channels, and functions.

```text
type Account enum {
    Personal struct { Name string }
    Business struct { Company string; Members []Account; Tags map[string]string }
}
a := Account.Business{Company: "Acme", Members: []Account{}, Tags: map[string]string{}}
match a {
case Personal(p): use(p.Name)
case Business(b): use(b.Members)
}
```

Emit a tag, typed payload fields, constructors, and ordinary Go switches:

```go
// Account must come from a variant constructor. Its zero value is invalid.
type Account struct {
    tag uint8
    personal struct { Name string }
    business struct { Company string; Members []Account; Tags map[string]string }
}
```

Tag zero is invalid. Constructors clear inactive fields. Replacing a variant replaces the value.
Reference fields keep Go aliases. Export the type name; keep representation fields private.
The layout uses the combined payload storage. It must pass the cost requirement below.

## 2. Checked constructors

Declare a construction rule once. Proposed syntax:

```text
type Quantity int where value > 0
q, err := NewQuantity(input)
if err != nil { return err }
use(q)
```

Generate a wrapper, constructor, and accessor. With the `errors` import:

```go
// Quantity must come from a successful NewQuantity call. Do not use its zero value.
type Quantity struct { value int }
func NewQuantity(value int) (Quantity, error) {
    if value <= 0 { return Quantity{}, errors.New("invalid quantity") }
    return Quantity{value: value}, nil
}
// Value requires a successfully constructed Quantity.
func (q Quantity) Value() int { return q.value }
```

The predicate runs only in the constructor. This is the business check the author requested.
`where` generates code; it is not a proof that every value satisfies the predicate.
Check the returned error before using the value. Ignoring it can break the rule, even in tgo.
Block direct tgo casts and field writes that bypass construction. Use `Value()` for arithmetic;
call the constructor again to produce a new Quantity. No automatic checks on reads or Go calls.

## 3. Explicit initialization

Keep Go structs and literals. Require initializers and every literal field.
`..default` explicitly fills omitted fields from declared defaults.

```text
type Request struct {
    ID string
    Tags map[string]string = map[string]string{}
}
a := Request{ID: "a", ..default} // A fresh map for this construction.
b := Request{ID: "b"} // Error: missing Tags.
var account Account // Error: no initializer or valid zero variant.
```

Keep Go mutation and copy rules. A struct copy shares its maps and slice data.
An explicitly supplied nil or zero is valid where the type permits it.
In tgo, enums and checked wrappers cannot be created by zero-filling allocation.
Use empty slices plus initialized values. Require presence tests for missing map entries.
Collection and default rules are defined in the [notes](../../research/go/tgo/notes.md).

## 4. Go is a trusted boundary

Import actual Go types. Emit direct calls both ways. Keep Go signatures, aliases,
callbacks, partial results, and errors. No boundary guards, copies, scans, or result wrappers.

```go
// Ordinary Go caller:
q, err := model.NewQuantity(3)
if err != nil { return err }
use(q)

bad := model.Quantity{} // Go permits this. It violates the documented contract.
use(bad)               // No generated validation will stop it.
```

Generate caller contracts like existing Go comments such as "do not pass nil".
Document required constructors, invalid zero values, error handling, and alias duties.
Go can break these contracts, including through a retained pointer or a shared collection.
Foreign values are trusted as declared. The same tradeoff exists between TypeScript and JavaScript.

Compile-time checks cover tgo declarations, initialization, fields, and match coverage.
They do not police the Go ecosystem. Agents, review, and tests check runtime contract use.
An invalid foreign value may cause wrong results or an ordinary Go panic; detection is not promised.

## 5. Review and cost

Review the type declarations, predicates, match branches, and Go caller contracts.
Test constructors, business transitions, defaults, and calls across the Go boundary.
Tests provide evidence; they do not turn the Go ecosystem into a sound type system.

Require the same runtime cost as the equivalent handwritten Go design.
A declared constructor check stays. Hidden read checks, validation wrappers, and copies do not.
Compare time, allocations, layout size, and generated calls. Reject costly layouts or helpers.
The enum layout and accessor calls still need this proof. No performance result is claimed yet.

[Rules, tradeoffs, and sources](../../research/go/tgo/notes.md).
