# Add checked struct construction

Status: Proposed

## Decision

Add a contextual `checked` marker after a struct declaration:

```tgo
type Port struct {
	Number int `json:"number"`
} checked

func (value Port) check() error {
	if value.Number <= 0 || value.Number >= 65536 {
		return fmt.Errorf("port %d is outside the valid range", value.Number)
	}
	return nil
}
```

Every checked struct must define one package-local `check() error` method on the checked type. A nil
result accepts the value. A non-nil result rejects it and becomes the construction error. The
compiler reports a missing or invalid method.

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

`Port{Number: number}` lowers to `PortValue{Number: number}.Port()`. The constructor creates the
opaque candidate and calls `candidate.check()` once. On failure, it returns the zero `Port` and the
same error.

Go callers use the same `PortValue` and constructor. Field tags, field defaults, generic type
parameters, and ordinary Go reference semantics are preserved. JSON unmarshaling validates before
it assigns the value.

## Value rules

The zero value of `T` is invalid. TGo code reads checked fields with normal selectors. The compiler
lowers those selectors to private payload reads. Go callers use `Value()`, which returns a copy of
the payload. Reference fields keep their normal aliases. A second checked struct can contain `T`,
so checked types form nominal chains without new syntax.

```tgo
type ServicePort struct {
	Port Port
} checked

func (value ServicePort) check() error {
	return nil
}
```

`tgolint` rejects direct wrapper construction, representation conversion, invalid zero use, and an
ignored construction error. The compiler only emits the representation and lowers the checked
literal.
