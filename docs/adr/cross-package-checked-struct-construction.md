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

The compiler emits one exported ABI method for each checked struct:

```go
func (value Port) TGoCheck() (Port, error) {
	return value.check()
}
```

The compiler lowers every checked literal, local or imported, to
`Port{Number: number}.TGoCheck()`. The method calls `check` once and returns its
value and error. A raw literal in the type's own `check` method keeps the
existing trusted exemption.

The receiver carries the type arguments for a generic checked struct, so the
same method shape applies without a separate generic helper.

`TGoCheck` is generated Go ABI, not TGo source API. TGo source cannot name or
call it, navigation does not expose it, and the compiler rejects a user method
with that reserved name. Generated fields keep their declared names and normal
Go visibility.

## Consequences

- Existing private fields remain private. Cross-package callers can omit them
  only when their zero value, a default, or `check` supplies valid state.
- Handwritten Go can still bypass validation with a direct literal. `tgolint`
  reports that bypass; a call to the generated ABI method performs validation.
- The compiler, specification, guides, navigation, hover, and syntax support
  must describe and recognize the same keyed literal form.
