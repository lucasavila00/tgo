# Add checked struct construction

Status: Proposed

## Decision

Add a contextual `checked` marker after a struct declaration. Every field must be private.
The type must define one package-local `check` method that returns the value and an error.

```tgo
type Port struct {
	number int
} checked

func (value Port) check() (Port, error) {
	if value.number <= 0 || value.number >= 65536 {
		return Port{}, fmt.Errorf("port %d is outside the valid range", value.number)
	}
	return value, nil
}
```

A nil error accepts the value returned by `check`. A non-nil error rejects construction. The
compiler reports a missing method or a method with another signature.

A checked struct literal is a fallible expression:

```tgo
port, err := Port{number: number}
port := Port{number: number}!
port := Port{number: number}!!
```

## Lowering

The compiler emits the ordinary Go struct and method. It removes only the `checked` marker.
It adds `.check()` after every composite literal of that checked type:

```go
port, err := Port{number: number}.check()
```

The compiler first records the checked struct names in the package. The expression pass then
rewrites their literals. It does not generate a wrapper, payload type, constructor, accessor, or
validation helper.

A field expression runs once in source order because the compiler does not copy or rebuild the
literal. `!` and `!!` keep their normal propagation behavior on the result of `check`.

## Rules

The private fields stop another package from constructing the value without `check`. Handwritten Go
in the declaring package can bypass the rule, so `tgolint` reports a direct checked literal in a Go
file. TGo source always rewrites one.

The zero value is invalid. Ordinary selectors read fields inside the declaring package. A checked
struct may contain another checked struct and therefore forms nominal chains without more syntax:

```tgo
type ServicePort struct {
	port Port
} checked

func (value ServicePort) check() (ServicePort, error) {
	return value, nil
}
```

A package exports its own fallible factory when callers in another package must construct a checked
value. No generated public constructor is part of this feature.
