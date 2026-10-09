# Add checked struct construction

## Decision

Add an opaque checked record with a public construction payload.

```go
newtype Port struct {
	Number int `json:"number"`
}

func (value PortValue) Validate() error {
	if value.Number <= 0 || value.Number >= 65536 {
		return fmt.Errorf("port %d is outside the valid range", value.Number)
	}
	return nil
}
```

A `newtype T struct` declaration creates the source payload name `TValue`. It
requires `func (TValue) Validate() error`. The fields are inputs. They are not
fields of `T`.

A checked literal has the static result type `(T, error)`:

```go
port, err := Port{Number: number}
port := Port{Number: number}!
port := Port{Number: number}!!
```

`!` and `!!` use their normal propagation rules. The validator error passes
through unchanged, so `errors.Is` and `errors.As` keep working.

## Generated Go API

The declaration emits:

```go
type PortValue struct {
	Number int `json:"number"`
}

type Port struct {
	value PortValue
}

func (value PortValue) Validate() error
func (value PortValue) Port() (Port, error)
func NewPort(value PortValue) (Port, error)
func (port Port) Value() PortValue
```

`Port{Number: number}` lowers to
`PortValue{Number: number}.Port()`. The helper calls `Validate` once. On
failure, it returns `Port{}` and the same error. On success, it stores the payload.

The source and generated names follow one rule: `T`, `TValue`, `NewT`, and
`TValue.T`. Go callers can discover and use the same payload and constructor.

## Value rules

`T` has nominal identity. Its fields remain private. `Value()` returns the
payload by value. Slices, maps, pointers, and other reference fields keep normal Go
aliasing. The wrapper prevents field reassignment; it does not deep-copy references.

The Go zero value of `T` is invalid. `Value()` returns a zero payload for it.
`tgolint` rejects use of an unconstructed zero. JSON marshals the payload and
unmarshal validates before assignment. A multi-field checked record has no automatic
text form.

Field tags and defaults follow normal TGo struct rules. Field expressions evaluate
once, from left to right, before validation.

Generic checked records keep their parameters:

```go
newtype Range[T cmp.Ordered] struct {
	Min T
	Max T
}
```

This emits `RangeValue[T]`, `Range[T]`, and `NewRange[T]`.

## Chaining

A second record can accept the first opaque value:

```go
newtype ServicePort struct {
	Port Port
}

func (value ServicePortValue) Validate() error {
	if value.Port.Value().Number == 22 {
		return errors.New("service port cannot be SSH")
	}
	return nil
}

port := Port{Number: number}!
service := ServicePort{Port: port}!
```

Each layer keeps its identity, payload, and validator. Construction of the outer
value cannot bypass construction of the inner value.

## Compiler and linter work

The compiler creates the payload and wrapper and lowers the checked literal. It does
not decide whether a result error was handled.

`tgolint` checks the validator signature, requires error handling or propagation,
rejects representation construction and conversion, rejects invalid zero use, and
applies the same rules to imported and generic checked records.
