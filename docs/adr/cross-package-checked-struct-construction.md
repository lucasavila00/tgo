# Keep checked literals across package boundaries

## Context

Checked literals are the TGo construction form, but private fields make an
imported literal fail Go visibility checks. Requiring a constructor call in
TGo would replace the current form and give the language two APIs. TGo needs
one construction form that works in every package and keeps the value opaque.

## Decision

Keep every checked-struct field private. The existing checked literal remains
the only TGo construction form, in the declaring package and in other packages:

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

```tgo
port, err := model.Port{number: number}
port := model.Port{number: number}!
port := model.Port{number: number}!!
```

An imported checked literal is construction, not ordinary field access. It can
name the imported checked type's private fields. This adds no syntax.

The declaring package generates this public Go ABI:

```go
func NewPort(number int) (Port, error) {
	return Port{number: number}.check()
}
```

The package-level name `New<Type>` is reserved for this generated ABI.

The compiler uses that ABI for checked literals. It builds the raw value and
calls the private `check` method exactly once. In TGo source, only the owning
`check` method can use a raw literal. That method can build and normalize its
local raw value.

Handwritten Go can call `NewPort`. TGo rejects a source-written call to the
generated function, navigation hides it, and TGo documentation shows only the
literal. The compiler generates no public field reader or setter.

Existing complete-literal and `..default` rules remain. Explicit field
expressions run once in source order. Selected defaults run once in field
order. `!` and `!!` keep their current error behavior.

Package-local code can read private fields. Cross-package reads stay private.
TGo rejects direct field writes and address-taking after construction, including
in the declaring package. Copying the value remains normal.

Reference-valued fields keep normal Go aliases. A caller can mutate shared data
through another alias, so checked structs do not provide deep immutability.

## Consequences

- TGo uses one literal API without new syntax in all packages.
- Go uses the generated constructor ABI.
- `tgolint` reports handwritten Go bypasses and unchecked constructor errors.
- The zero value remains invalid and subject to the existing use checks.
