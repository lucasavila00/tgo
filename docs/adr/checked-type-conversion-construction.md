# Use conversion syntax for checked types

## Decision

Replace the `where` clause as the long-term checked-type syntax. Declare the base
type and a reserved validator method with ordinary Go syntax. Construct a value with
a conversion-shaped call.

```go
type Port int

func (Port) Valid(value int) bool {
	return value > 0 && value < 65536
}

func PortNumber(text string) (int, error) {
	number := strconv.Atoi(text)!
	port := Port(number)!
	return port.Value(), nil
}
```

TGo treats `func (T) Valid(value Base) bool` as the checked-type declaration. The
receiver has no name because validation applies to the proposed base value. The
method name, parameter count, parameter type, and result type are fixed.

This option is recommended. The construction reads like a Go conversion, the type
name is the constructor name, and the validator is normal Go syntax.

## Generated Go API

TGo keeps the opaque representation and emits the current Go API:

```go
type Port struct { value int }

func NewPort(value int) (Port, error)
func (value Port) Value() int
```

A TGo call to `Port(number)` lowers to `NewPort(number)`. A following `!` or
`!!` uses the normal propagation rule. A caller can also receive both results:

```go
port, err := Port(number)
```

The generated constructor returns `invalid Port` when `Valid` returns false.
The validator runs once during construction.

## Type rules

`Port` has nominal identity. It is not assignable to `int`. `Value` is the only
representation accessor. The zero value remains invalid and is only an error
sentinel from failed construction.

A checked type can extend another checked type without exposing its representation:

```go
type ServicePort Port

func (ServicePort) Valid(value Port) bool {
	return value.Value() != 22
}
```

`ServicePort(port)!` first requires a valid `Port`. Its `Value` method returns
`Port`. Each validator owns one rule layer.

## Compiler and linter work

The parser already accepts the declaration and method. The compiler must recognize
the exact validator shape, emit the opaque wrapper, and lower construction calls.
Imported checked types need exported analysis metadata so the compiler can
distinguish a checked conversion from a Go conversion. This is a medium codegen
change. It removes the predicate mini-language and its expression capture.

`tgolint` must reject:

- a validator with the wrong signature;
- checked-type literals, `new(T)`, and representation conversions;
- construction whose error result is discarded; and
- a derived type that omits its own validator.

The linter can use the TGo syntax model and type facts. It does not need to inspect
generated Go. The compiler only emits the wrapper and lowers the explicit
construction expression.
