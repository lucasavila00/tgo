# Consider struct-like checked construction

## Decision

Do not use a public struct literal as the checked-type constructor.

The proposed form uses only familiar Go declarations and adds validation at the
literal:

```go
type Port struct {
	Value int
}

func (port Port) Valid() bool {
	return port.Value > 0 && port.Value < 65536
}

func PortNumber(text string) (int, error) {
	number := strconv.Atoi(text)!
	port := Port{Value: number}!
	return port.Value, nil
}
```

The shape is easy to read. It also matches enum payload construction. It does not
preserve the checked-type invariant.

## Generated Go API

The direct output would be:

```go
type Port struct {
	Value int
}

func (port Port) Valid() bool
```

TGo would lower `Port{Value: number}!` to a helper that evaluates the literal,
calls `Valid`, and returns `(Port, error)`. Validation failure would return
`Port{}` and `invalid Port`. The normal `!` and `!!` rules would propagate
that error.

This API lets any Go or TGo caller write `Port{Value: -1}` or mutate
`port.Value`. The constructor is visible, but it is optional.

## Alternative field rewrite

The compiler could emit a private field and a `Value()` accessor:

```go
type Port struct { value int }
func (port Port) Value() int
```

That rewrite makes source fields differ from generated fields. Reflection, JSON
tags, field selection, method bodies, and Go callers would see a different type.
Embedding another checked struct would add the same mismatch. This is too much
codegen for a small construction feature.

## Type rules

The struct has nominal identity. Its zero value is invalid. Composition through
embedding exposes the embedded value and its fields, so it does not provide safe
extension. A second validator can check the outer value, but callers can still
mutate either layer.

A validator can return `error` instead of `bool` to keep a detailed failure:

```go
func (port Port) Validate() error
```

That is useful, but it does not close construction or mutation paths.

## Compiler and linter work

The parser accepts all declarations. The compiler would lower a composite literal
followed by `!` or `!!`. The generated helper must preserve literal evaluation
order.

`tgolint` would have to reject every unchecked literal, field assignment,
pointer mutation, reflection path, and generic write. It could enforce those rules
in TGo source, but ordinary Go callers could still bypass them.

Reject this option. It is suitable for validation at an API boundary. It is not a
checked newtype. The conversion-shaped proposal keeps one opaque value and a clear
construction path with less analysis.
