# Use one checked struct literal across packages

## Context

Checked structs now require private fields and lower each literal to a private
`check` method. Another TGo package cannot set those fields or call that method.
An exported factory gives the same type a second construction form and makes
each package author define language support by hand.

## Decision

Use a keyed checked-struct literal as the only TGo construction form, in the
declaring package and in other TGo packages.

Checked structs can contain exported and unexported fields. Field names keep
normal Go visibility in literals and selectors. Code in another package can set
and select only exported fields.

```tgo
// package model
type Port struct {
	Number int
} checked

func (value Port) check() (Port, error) {
	if value.Number < 1 || value.Number > 65535 {
		return Port{}, ErrInvalidPort
	}
	return value,
}
```

```tgo
// package server
port, err := model.Port{Number: number}
```

All checked-struct literals must use field keys. Positional literals are
invalid in all packages. An embedded field uses its Go field name and normal
visibility. `..default` fills omitted defaulted fields before validation.

The compiler emits one exported helper for each checked struct:

```go
func TgoCheckPort(value Port) (Port, error) {
	return value.check()
}
```

The compiler lowers every checked literal, local or imported, to that helper.
The helper calls `check` once and returns its value and error. A raw literal in
the type's own `check` method keeps the existing trusted exemption.

For a generic checked struct, the helper repeats the type parameters and their
constraints. Its shape is
`func TgoCheckBox[T Constraint](value Box[T]) (Box[T], error)`. Normal Go type
inference supplies the helper arguments at a lowered literal.

`TgoCheck<Type>` is generated Go ABI and is reserved by the compiler. It is not
a second TGo source API. Generated fields keep their declared names and normal
Go visibility.

## Consequences

- Existing private fields remain private. Cross-package callers can omit them
  only when their zero value, a default, or `check` supplies valid state.
- Handwritten Go can still bypass validation with a direct literal. `tgolint`
  reports that bypass; a call to the generated helper performs validation.
- The compiler, specification, guides, navigation, hover, and syntax support
  must describe and recognize the same keyed literal form.
