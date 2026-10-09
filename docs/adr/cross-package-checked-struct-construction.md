# Generate one checked struct constructor

## Context

A checked struct is an opaque value whose usable instances passed validation.
Private fields protect its representation, but the current literal form cannot
cross a package boundary. A project-defined factory gives different packages
different construction APIs and does not make the language enforce one path.

## Decision

Keep every checked-struct field private. Generate one exported fallible
constructor as the only construction API for TGo and Go code:

```tgo
type Port struct {
	number int
} checked

func (value Port) check() (Port, error) {
	if value.number < 1 || value.number > 65535 {
		return Port{}, ErrInvalidPort
	}
	return value,
}
```

```go
func NewPort(number int) (Port, error) {
	return Port{number: number}.check()
}
```

Code in the declaring package, another TGo package, or a Go package calls
`NewPort`. TGo error propagation applies to that call without special syntax.

The generated signature follows these rules:

- Its name is `New<Type>` and that package-level name is reserved.
- It has one parameter for each nonblank field, in declaration order.
- A named field gives the parameter its name and type.
- An embedded field gives the parameter its declared field name and type.
- A blank field has no parameter and keeps its zero value.
- A generic constructor repeats the type parameters and constraints of the
  checked struct.

Checked structs cannot declare field defaults. One fixed function cannot both
accept an override and omit the same argument without another API or an options
mechanism. Approval of this decision includes this restriction.

The constructor evaluates its arguments once, builds the private value, calls
the package-local `check` method once, and returns its value and error. It does
not generate readers, setters, or another validation entry point.

TGo rejects a checked-struct literal outside its own `check` method. It also
rejects field assignment after construction, including in the declaring
package. The private `check` method can build and normalize its local raw value.
Package-local code can read private fields, and copying a validated value keeps
normal Go value semantics.

## Consequences

- Existing checked literals must become `New<Type>` calls.
- Other packages can construct the value but cannot access its representation.
- Handwritten Go in the declaring package can still bypass these rules, so
  `tgolint` reports raw literals, field writes, and unchecked constructor
  errors.
- The zero value remains invalid and subject to the existing use checks.
