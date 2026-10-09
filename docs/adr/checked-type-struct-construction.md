# Add checked struct construction

Status: Proposed

## Decision

Extend the existing checked type syntax. Do not add `newtype`.

```tgo
type Port struct {
	Number int `json:"number"`
} where value.Validate()

func (value PortValue) Validate() error {
	if value.Number <= 0 || value.Number >= 65536 {
		return fmt.Errorf("port %d is outside the valid range", value.Number)
	}
	return nil
}
```

A `where` expression must return `bool` or `error`. A Boolean result accepts the value when it
is true. When it is false, construction returns the zero checked value and the generated
`invalid Port` error. An error result accepts the value when it is nil. When it is non-nil,
construction returns the zero checked value and that error unchanged.

A checked struct literal is a fallible expression:

```tgo
port, err := Port{Number: number}
port := Port{Number: number}!
port := Port{Number: number}!!
```

The literal evaluates each field once in source order. `!` and `!!` keep their normal error
propagation behavior.

## Generated Go API

For a checked struct `T`, the compiler exports its construction payload and keeps `T` opaque:

```go
type PortValue struct {
	Number int `json:"number"`
}

type Port struct { value PortValue }

func NewPort(value PortValue) (Port, error)
func (value PortValue) Port() (Port, error)
func (port Port) Value() PortValue
```

`Port{Number: number}` lowers to `PortValue{Number: number}.Port()`. The constructor evaluates
the `where` expression once. On failure, it returns the zero `Port` and the validation error.

Go callers use the same `PortValue` and constructor. Field tags, field defaults, generic type
parameters, and ordinary Go reference semantics are preserved. JSON unmarshaling validates before
it assigns the value.

## Value rules

The zero value of `T` is invalid. `Value()` returns a copy of the payload. Reference fields keep
their normal aliases. A second checked struct can contain `T`, so checked types form nominal chains
without new syntax.

```tgo
type ServicePort struct {
	Port Port
} where value.Validate()
```

`tgolint` rejects direct wrapper construction, representation conversion, invalid zero use, and an
ignored construction error. The compiler only emits the representation and lowers the checked
literal.
