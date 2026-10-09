# Generate checked struct constructors

Status: proposed

## Decision

For each checked struct `Type`, TGo generates one exported package function
named `NewType`. `Type` is the exact declared type name.

The constructor has one parameter for each non-blank field. Parameters keep
field declaration order. A field group such as `left, right int` produces two
parameters in that order.

A named field gives its name and type to its parameter. An embedded field uses
its Go field name and declared type. A field with a default still produces a
required parameter. If a parameter name conflicts with a type parameter,
TGo adds `_1`, `_2`, and so on until the name is unique.

The constructor returns `(Type, error)`.

For example:

```text
type Port struct {
    number int
} checked
```

TGo generates this public signature:

```go
func NewPort(number int) (Port, error)
```

A generic checked struct copies its type parameter list to the constructor and
applies the type arguments to its result:

```text
type Entry[K comparable, V any] struct {
    key   K
    value V
} checked
```

TGo generates this public signature:

```go
func NewEntry[K comparable, V any](key K, value V) (Entry[K, V], error)
```

## Compatibility

`NewType` is reserved for the generated constructor. An existing package
declaration with that name is a compile-time collision at the checked struct.

Implementation is deferred until this ADR is approved.
