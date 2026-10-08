# Go with small additions

A Go superset. Keep existing source and behavior. Add short forms for repeated code.
Keep Go packages and libraries. Change one feature at a time.

## 1. Short error branches

Call once. On error, run the branch. End it with `return`.
Keep normal wrapping, custom errors, partial results, and defers.

```text
value := loadCustomer(id) else err {
    return nil, fmt.Errorf("load customer %s: %w", id, err)
}
// Equivalent Go; callErr is a fresh internal name:
value, callErr := loadCustomer(id)
if callErr != nil {
    return nil, fmt.Errorf("load customer %s: %w", id, callErr)
}
```

Start with `(T, error)` calls and one new value name. The error name exists
only in the branch. Keep the code in the same function. Savings: binding and test.

## 2. Restricted interface values

Allow only the listed concrete types. Keep `nil` as the zero value.
Use existing structs and type switches. Based on [Go proposal #57644][unions].

```text
type Personal struct { Name string }
type Business struct { Name, Company string }
type Account interface { Personal | Business }
var account Account // nil.
account = Business{Name: "Ana", Company: "Acme"}
account = 42 // Error.
```

Deferred: ordinary Go cannot use this interface as a value type.
Emitting `any` loses checks at Go calls. A wrapper changes interface behavior.
Resolve that boundary first. Fields can still be empty; switches can still miss cases.

## 3. Checked constructors

Generate the wrapper and constructor. Keep existing Go type rules.
Proposed tool and its output:

```text
//go:generate checkedtype Quantity int "value > 0"
```

```go
type Quantity struct { value int }
func NewQuantity(value int) (Quantity, error) {
    if value <= 0 { return Quantity{}, errors.New("invalid quantity") }
    return Quantity{value: value}, nil
}
func (q Quantity) Value() int { return q.value }
```

`var q Quantity` still creates zero. Package code can still change the field.
This saves constructor code. It does not prove that every value is valid.

## 4. Field getters

Turn a field name into a function. Copy the struct value.
Start with direct exported fields on named, non-generic struct types.

```text
getCompany := Business.Company
// Equivalent Go:
getCompany := func(b Business) string { return b.Company }
```

Keep method expressions and access rules. Exclude pointer and promoted fields first.
Zero input gives an empty string. Pointer, slice, or map fields still share their data.

## 5. Slice comprehensions

Filter and map one slice. Use a typed result expression.
Based on [Elixir comprehensions][comprehensions].

```text
amounts := [for _, sale := range sales if sale.Paid: sale.AmountCents]
// Equivalent Go; AmountCents is int64; input and tmp are fresh names:
input := sales
var tmp []int64
for _, sale := range input {
    if sale.Paid { tmp = append(tmp, sale.AmountCents) }
}
amounts := tmp
```

Evaluate input once. Filter before the result expression. Keep input order.
No matches: nil slice. Keep Go range behavior. Grouping stays in library functions.

Start with error branches. Compare source size and review effort against plain Go.
Add getters and comprehensions separately. Test effects and Go calls both ways.
[Work and sources](notes.md).

[unions]: https://github.com/golang/go/issues/57644
[comprehensions]: https://hexdocs.pm/elixir/comprehensions.html
