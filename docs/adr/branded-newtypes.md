# Add explicit newtypes

## Decision

Add one `newtype` declaration. It covers nominal labels and checked scalar values
without tag structs or generated constructor names in TGo source.

```go
newtype Port int

func (Port) Valid(value int) bool {
	return value > 0 && value < 65536
}

func (Port) Parse(text string) (int, error) {
	return strconv.Atoi(text)
}

newtype UserID string
```

A missing `Valid` method accepts every base value. A method must have the exact
form `func (T) Valid(Base) bool`. An optional parser has the exact form
`func (T) Parse(string) (Base, error)`. TGo reserves both methods.

Use braces for literals and conversion syntax for dynamic values:

```go
http := Port{80}
https := Port{"443"}
user := UserID{"u-123"}

number := strconv.Atoi(text)!
port := Port(number)!
parsed := Port.Parse(text)!
```

`T{literal}` is a single-value literal constructor. `tgolint` must parse or
convert the constant to the base type and prove `Valid`. A numeric newtype may
accept a quoted numeric literal. Other cross-type literals are errors.

`T(expression)` returns `(T, error)` when `T` has a predicate. `!` and `!!`
use their normal rules. It returns one `T` for a nominal newtype with no predicate.
`T.Parse(text)` always returns `(T, error)`.

## Lowering and Go API

For `Port`, TGo emits:

```go
type Port struct { value int }

func NewPort(value int) (Port, error)
func MustPort(value int) Port
func ParsePort(text string) (Port, error)
func (value Port) Value() int
```

`Port{80}` lowers to `MustPort(80)`. `Port(number)` lowers to
`NewPort(number)`. `Port.Parse(text)` lowers to `ParsePort(text)`. These rules
also apply to `domain.Port`; imported TGo metadata records the base, predicate,
and generated names.

`NewPort` returns `Port{}` and `invalid Port` when the predicate is false.
`MustPort` calls `NewPort` and panics on failure. TGo literal checks make that
panic unreachable in checked source. `ParsePort` calls the reserved parser, then
calls `NewPort`. Without a parser, strings use identity and numbers use `strconv`
with the exact base width. Other bases have no generated `ParseT`.

The base is an exact Go type. Use `int8`, `uint16`, `float64`, or a named type
when that width or method set matters. Newtypes do not inherit base operators.
Call `Value()` before arithmetic.

## Identity, composition, and zero

Each declaration has nominal identity. Two newtypes with the same base are not
assignable. Extension uses another newtype as the base:

```go
newtype ServicePort Port

func (ServicePort) Valid(value Port) bool {
	return value.Value() != 22
}
```

`NewServicePort` first rejects an invalid `Port`, then applies its own predicate.
`ServicePort.Value()` returns `Port`. This is composition, not subtyping.

The Go zero value contains the base zero. It is valid when all predicates accept
that value and invalid otherwise. `Value` still returns the stored zero. `tgolint`
must reject use of an invalid zero as a constructed value.

Generic declarations use Go type parameters:

```go
newtype ID[Entity any] string
id := ID[User]{"u-123"}
```

The output is `ID[Entity]`, `NewID[Entity]`, `MustID[Entity]`, and
`ParseID[Entity]`. Callers give type arguments when Go cannot infer them.

## JSON, text, and lint rules

JSON marshals the base value. JSON unmarshal decodes the base and calls `NewT`.
Text marshal uses the base text form. Text unmarshal calls `ParseT`. Predicate,
syntax, range, and parse failures pass through. An invalid zero returns
`invalid T` from marshal.

`tgolint` owns the proof rules. It must:

- evaluate literal conversion, range, and the supported constant predicate subset;
- require dynamic syntax when it cannot prove a literal;
- require callers to use both results or `!` or `!!`;
- reject representation literals, conversions, `new(T)`, and invalid zero use; and
- apply the same rules to generic and imported newtypes.

The compiler only emits the wrapper and helpers and lowers the three explicit
construction forms. It does not run the policy checks.
