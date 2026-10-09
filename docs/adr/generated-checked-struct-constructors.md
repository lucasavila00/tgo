# Generate checked struct constructors

Status: proposed

## Context

A checked struct has private fields and a `check` method. A TGo literal calls
that method and returns the checked value and its error. Another package cannot
write the literal because it cannot select the private fields. The declaring
package must now write an exported fallible factory for each cross-package use.
Projects can therefore use different names, parameter orders, and validation
paths for the same operation.

TGo can generate one factory for each checked struct. This addition creates a
public API, so its name, signature, ownership, and compatibility rules must be
stable before implementation starts.

## Decision

For each checked struct `Type`, generate one package function named `NewType`.
`Type` is the exact declared type name. Thus, `Port` generates `NewPort`, and
`port` generates `Newport`.

The function has one parameter for each non-blank declared field. It keeps the
flattened field declaration order. A group such as `left, right int` becomes
two parameters in that order. A field named `_` stays at its zero value because
Go cannot store an observable value in it. Fields with defaults are also
required parameters. Go has no optional parameters, and a second constructor
shape would make the API depend on field defaults.

A named field gives its name to its parameter. An embedded field uses its Go
field name. For example, `*item` uses `item`, and `box[T]` uses `box`. A blank
field has no parameter. If a parameter name conflicts with a type parameter or
an earlier generated parameter, append `_1`, `_2`, and so on until the name is
unique. This rule changes names only when Go cannot use the first choice.

The result is `(Type, error)`. The generated body initializes the fields in
declaration order and calls `check` exactly once:

```go
func NewPort(number int) (Port, error) {
	return Port{number: number}.check()
}
```

The compiler must use one direct `check` call in emitted Go. It must not lower
the constructor through a second validation path. Function arguments use Go's
normal evaluation rules before the body starts.

An embedded field stays one constructor parameter with its declared type. TGo
does not flatten promoted fields. An unexported embedded type can make the
exported constructor difficult or impossible to call from another package.
The generator does not replace that type with a different representation. A
project that needs a different boundary can add a separate named factory.

A generic checked struct copies its type parameter list to the function and
applies each type parameter to the result and literal:

```go
type Pair[K comparable, V any] struct {
	key   K
	value V
} checked

func NewPair[K comparable, V any](key K, value V) (Pair[K, V], error) {
	return Pair[K, V]{key: key, value: value}.check()
}
```

Normal Go type inference applies. A caller must give explicit type arguments
when the parameter values do not provide enough information.

The generated name is reserved in the active package. Any package declaration
named `NewType`, including a declaration in a `.go` file or another generated
API, is a collision. The compiler reports the collision at the checked struct
name. It does not silently omit the constructor and does not select a different
name.

Calls to the generated constructor have their normal generated function hover.
Go to definition maps the function to the checked struct name. Find references
uses the constructor object and maps its declaration to the same source range.
Document and workspace symbols include a generated `NewType` function symbol.
Its full range is the checked struct declaration, and its selection range is
the checked struct name. Hover on the struct name continues to show the struct,
not the generated function.

Checked struct literals remain supported. They keep their current fallible
behavior and validation rules. The constructor is the consistent public
boundary, not a replacement for same-package literals. Direct Go literals
remain outside the TGo guarantee, and `tgolint` continues to report them.

## Compatibility

This change adds one exported package name for every checked struct. A project
that already declares `NewType` must remove it, rename it, or rename the checked
struct before it can use a compiler version that implements this decision.
Other manual factory names remain valid. TGo does not redirect them to the
generated constructor and cannot prove that they validate exactly once.

The strict collision rule is an intentional source compatibility break. It
keeps one API for all checked structs and makes generated ownership clear.

## Alternatives

### Use `MakeType` or a type-qualified name

`MakeType` is also valid Go style, but `NewType` is more common for a fallible
constructor that returns a new value. Go does not let a type own a package-level
function namespace, so `Type.New` would need a receiver or another generated
value. `New` plus the exact type spelling is simple and deterministic.

### Keep project-defined factories

This option has no compatibility cost. It also keeps the current inconsistent
names, parameter orders, and validation paths. It does not meet the goal of one
normal construction API.

### Suppress generation for a compatible `NewType`

This option would preserve many current factories. The compiler would have to
define signature compatibility and trust a body that it does not control. It
could not guarantee one validation call, and navigation ownership would differ
between packages. The decision rejects this option.

### Generate a method or a builder

A method needs a receiver before construction. A builder adds another type,
more calls, and more zero-value rules. Neither form is needed for a function
that sets all fields once.

### Use a configuration value

A public options struct could hide parameter order and make default fields
optional. Its fields would expose the checked struct representation or require
another generated naming system. It also adds allocation and alias questions
that a direct function signature avoids.

### Use sorted, positional, or renamed field parameters

Sorting fields would disconnect the call from the declaration. Names such as
`value1` would avoid name conflicts but would make hover signatures less useful.
Declaration order and source field names keep the generated signature close to
the source. The suffix rule handles the small set of real conflicts.

### Flatten embedded fields

Flattening could make some constructors easier to call across packages. It
cannot preserve the identity of embedded pointers, interfaces, or named types,
and promoted names can conflict. One parameter for the embedded value keeps Go
semantics.

### Reject generic checked structs

This option would reduce compiler work, but it would make checked construction
unavailable for a normal Go type form. Copying the type parameters and applying
them to `Type` preserves the source constraints and uses Go inference.

### Reuse checked literal lowering in the generated body

This option could reduce emitter code, but it could also add a second `check`
call if the generated body already contains validation. The output contract is
one visible direct call. The implementation can share internal code only if the
emitted constructor still contains exactly one call.

### Keep generated locations in the Go output

Generated Go locations would make definition results leave the TGo source and
would expose files that editor settings can hide. Mapping the constructor to
the checked struct gives it a stable source owner. A separate function symbol
keeps `NewType` available in symbol searches without changing hover on `Type`.

### Select another name after a collision

A suffix such as `NewType2` would keep compilation possible, but the public API
would depend on unrelated package declarations. A source diagnostic keeps the
constructor name predictable.

### Prohibit checked struct literals

This option would force all calls through `NewType`. It would break current TGo
source and remove concise same-package construction without adding validation.
Both forms can use the same `check` method exactly once.

## Implementation status

This ADR contains no implementation. Production code, generated files, the
README, the specification, guides, and tests stay unchanged. Implementation is
deferred until this ADR is approved.
