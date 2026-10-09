# Add branded string and number formats

## Decision

Add branded formats as a separate feature for nominal labels that do not need a
runtime predicate. Do not use them as a replacement for checked types.

The source form uses Go generic types and marker types:

```go
type EmailTag struct{}
type UserIDTag struct{}

type Email = StringFormat[EmailTag]
type UserID = IntFormat[UserIDTag]

email := StringFormatOf[EmailTag](text)
id := IntFormatOf[UserIDTag](number)
```

`Email` and `UserID` cannot be mixed with plain values or with another brand.
The constructor is explicit and searchable. No hidden name is generated from the
alias.

This design follows branded `StringFormat<Tag>` and `NumberFormat<Tag>` types.
Use it for units, identifiers, parsed formats, and trusted boundary data.

## Generated Go API

TGo can provide ordinary generic wrappers:

```go
type StringFormat[Tag any] struct { value string }
type IntFormat[Tag any] struct { value int }

func StringFormatOf[Tag any](value string) StringFormat[Tag]
func IntFormatOf[Tag any](value int) IntFormat[Tag]
func (value StringFormat[Tag]) Value() string
func (value IntFormat[Tag]) Value() int
```

The number base needs separate `IntFormat`, `UintFormat`, and `FloatFormat`
families. One `NumberFormat` cannot preserve all Go numeric widths and operations.

Construction cannot fail. Error propagation belongs in the parser or checker that
earns the brand:

```go
func ParsePort(text string) (IntFormat[PortTag], error) {
	number := strconv.Atoi(text)!
	if number <= 0 || number >= 65536 {
		return IntFormat[PortTag]{}, errors.New("invalid port")
	}
	return IntFormatOf[PortTag](number), nil
}
```

## Type and zero rules

Each tag argument gives a distinct concrete type. An alias has the identity of its
instantiation. An unexported tag limits use of its name to its package, but it does
not add validation.

The zero value contains the zero base value. It is valid as a brand because the
feature has no predicate. Code that needs an invalid zero must use a checked type or
an enum.

Brands compose with a marker that embeds other markers:

```go
type NormalizedEmailTag struct {
	EmailTag
	NormalizedTag
}

type NormalizedEmail = StringFormat[NormalizedEmailTag]
```

This is explicit but verbose. TGo should not add TypeScript-style intersections or
string value parameters in the first version.

## Compiler and linter work

A library implementation needs no parser change and little code generation. The
compiler only needs generated generic wrappers if they are built-ins. A normal TGo
package can test the design first.

`tgolint` should prevent composite literals and representation conversions for the
generic wrappers. It should allow construction only through `StringFormatOf`,
`IntFormatOf`, `UintFormatOf`, and `FloatFormatOf`. It cannot prove a semantic
predicate because none exists.

Accept branded formats for tag-only nominal types. Reject them for `Port` range
checks. The conversion-shaped checked type remains the better runtime validation
design.
