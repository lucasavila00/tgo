# tgo rules and review cases

## Initialization

Source declarations need initializers. Struct literals need every field or `..default`.
The marker uses declared defaults only. It does not invent a default for a required field.
An explicit zero or nil is still an initializer. Initialization is not a content constraint.

Evaluate explicit field expressions in written order. Then evaluate selected defaults
in field declaration order. Evaluate each once. Defaults cannot refer to the value being built.
A default may be a literal, function call, or reference to existing data.
A literal map or slice creates new storage per construction. A reference keeps its aliases.
Skip the default expression when the caller supplies that field.

```text
type Request struct {
    ID string
    Labels map[string]string = map[string]string{}
}
a := Request{ID: "a", ..default}
b := Request{ID: "b", ..default}
c := a
c.Labels["x"] = "y" // Visible through a, not b.
```

Keep source assignment and mutation. Do not add implicit deep copies or ownership transfers.
Changing an enum variant replaces the entire value. A match binds a copy of its active payload.
Field writes on that copy follow Go value/reference rules; maps and slices can still share data.

## Checked constructors

`type Quantity int where value > 0` declares a wrapper and a constructor rule.
Emit a private value field, `NewQuantity(int) (Quantity, error)`, and `Value() int`.
It is not a Go integer type with unrestricted casts and arithmetic.
tgo disallows writes or conversions that bypass the constructor. Arithmetic uses the base value;
construct the result again when a Quantity is needed.

The predicate runs at construction only. Constructor failure returns a zero wrapper and an error.
The caller must check that error. Ignoring it can break the invariant inside tgo as well as Go.
Do not claim a global refinement-type proof. Go callers can also make zero directly.
This check is declared business logic, not FFI validation or hidden runtime enforcement.

Generate comments on exported types, constructors, accessors, and business functions.
State invalid zero values, construction requirements, error-result use, and alias obligations.
Agents and tests should check that callers follow the comments. This is not a soundness theorem.

## Which Go zero values are valid?

- Scalars: valid.
- Pointers, slices, maps, channels, functions, and interfaces: nil is valid.
- Structs: valid only if every field has a valid zero.
- Arrays: valid only if every element has a valid zero. A zero-length array is valid.
- Enums: invalid. Tag zero is reserved.
- Checked wrappers: no implicit zero. Use their constructor, even if the predicate accepts zero.

These are value rules, not field-default rules. No enum has an implicit first variant.
A nil slice of enums has no elements and is valid. A non-empty zero-filled slice is not.

`make([]T, n)` explicitly requests zero elements. Accept it when T has a valid zero.
Otherwise require n to be the constant 0; capacity may be dynamic.
`new(T)` requests the zero of T. Check T, not the returned pointer type.
Neither operation evaluates declared defaults. A zero-valid struct can be allocated this way,
even when its literal requires named fields. This is an explicit request for all Go zeros.

Reject array or slice literals with holes. `[]T{}` is empty, not one omitted element.
A fixed-length array literal must supply every index. Use explicit zero allocation where valid.
For generic code, each zero-producing operation must be valid for every allowed type argument.
A generic Go function may still produce an invalid foreign value. The caller contract forbids it;
the generated code does not detect it on read.
Named result slots need explicit assignment before any read or bare return, for all types.
Their signature declarations do not count as initialization.

## Collection operations

For element types whose zero is invalid:

- `make([]T, 0, cap)` reserves storage; no zero element is readable.
- `append` adds supplied valid values.
- Reslicing may not increase the current length. Known violations are compile errors.
  Accept a dynamic bound only when source control flow already proves it is within len.
  Otherwise reject it at compile time. Do not insert a new runtime guard.
- `clear(slice)` is a compile error. It would create invalid elements.
- Element writes, `copy`, iteration, and shortening remain available.

For zero-valid element types, keep Go allocation, reslicing, and clearing rules.
Keep bounds panics and nil behavior. No automatic conversion of a nil map to an empty map.
Map assignment requires valid keys and values. `delete` and `clear(map)` remove entries.

Use native Go `copy` and `append`. Do not scan elements before or after the operation.
Type-check source arguments. Trust foreign values. Preserve overlapping-slice behavior.
No hidden copy, ownership transfer, or alias tracking runs at runtime.

## Missing values

A map miss, closed-channel receive, or failed assertion can return an invalid zero.
For such result types, require the ordinary Go presence test:

```text
if account, ok := accounts[id]; ok {
    use(account)
} else {
    // account is not in scope here.
}
```

The compiler binds the model value only in the successful branch. There is no tag check.
A foreign map can still contain an invalid enum at an existing key; this breaks the contract.
Changing an unrelated bool cannot establish presence for the source type checker.
Plain reads are allowed when the result zero is valid. Channel range stops at close.
Received values are trusted. Nil channels retain their blocking behavior.

## Enum storage and equality

Use a tag plus separate typed Go payload fields. Do not overlay memory with unsafe unions.
Go's garbage collector must see references in every payload field.
Only the active payload is a source value. Constructors zero inactive storage.
Inactive references must not remain after variant replacement.
This layout can use more space than a Rust enum. Stack allocation is not guaranteed.

References break recursive size dependencies. `Children []Node` and `Next *Node` are valid.
An enum directly containing itself by value has infinite size and is rejected.
Check source constructions statically. Emit no inline or referenced graph validation.
Reference cycles keep ordinary Go behavior.

Use ordinary Go equality for comparable generated structs. Source constructors keep inactive
fields zero, so valid source values compare by tag and active payload. All payloads must be
comparable. A slice or map field makes the enum non-comparable. Interface fields keep Go's
dynamic comparability rules. No custom equality helper or validity scan is inserted.
Foreign code that violates the representation contract also voids these source assumptions.

## Trusted Go FFI

Source typing ends at foreign code. Go calls and values are trusted, as JavaScript values
are trusted across a TypeScript boundary. This is a deliberate loss of enforcement.
Type-check signatures, not runtime values. No separate safe or unsafe FFI modes are needed.

Use native Go calls and references. Preserve aliases, overlapping slices, interior pointers,
pointer keys, cycles, nil, callbacks, and object identity. Keep named Go type identity.
Support functions, methods, interfaces, channels, variadic calls, and generic calls.
Do not serialize, clone, validate, or copy values merely because they cross the boundary.

Export model type names with private representation fields. Go must be able to name these
types in signatures, fields, containers, and callbacks. Go can still construct zero values.
Generated constructors set fields. Accessors expose payloads as the source operation requires.
Source metadata maps variants to those accessors. No accessor performs hidden model validation.
An explicit variant test still performs the requested tag comparison.

Go callbacks use their original signatures and Go-managed lifetime. No error result is added.
Keep the original result tuple, partial results, typed nil errors, and error identity.
Capture and forward foreign values as ordinary Go values. Remove the foreign-slot mechanism.
Do not generate ffi.InvalidValue panics or validation wrappers.

```text
a, err := legacy.LoadAccount()
if err != nil { return err }
use(a) // Trust the declared Go result type.
```

An invalid foreign value may cause a normal Go panic or wrong program behavior.
There is no guarantee that the error is detected. A match uses a normal tag switch.
Its unmatched default may panic; there is no preceding read or entry check.
Do not use violated source assumptions to justify unsafe memory operations in emitted Go.

## Aliases and initialization history

A Go caller can make a zero enum through a literal or generic code. Private fields do not
prevent it. A retained Go alias can replace a previously valid value after a call.

```go
func Wipe[T any](items []T) { clear(items) }
```

Wipe can zero a shared enum slice. Source reads then receive those zeros without validation.
The caller broke the source model contract. An int slice remains valid after the same operation.
Go can likewise modify a captured local through a pointer. No alias monitor is generated.

Go cannot report which fields the caller explicitly initialized. Enforce initializer rules
in source code only. Treat foreign fields as supplied values, regardless of how Go created them.
Retained callbacks and channels use the same trust rule. Keep normal Go synchronization duties.
There is no new race, ownership, purity, or foreign-code correctness guarantee.

## Cost requirement

No runtime cost beyond the equivalent handwritten Go design. In particular, add no automatic
validation, element scans, deep copies, special result slots, or boundary wrappers.
Keep ordinary Go bounds checks, map operations, garbage collection, and requested allocations.
Static restrictions must not secretly turn into runtime guards when proof fails.

The tagged-struct layout still has costs: combined payload storage, Go value copies,
and accessor calls across package boundaries. Inlining is not guaranteed.
These are open cost questions, not proof that the design meets the requirement.
Reject a layout or helper if it loses to the Go alternative. Do not weaken the requirement
by comparing only with an unnecessarily expensive handwritten implementation.

## Evidence to compare

Use the same business task in Go and the proposed language. Count source and adapter code.
Compare review effort, execution time, allocation, and payload storage costs.
Include shared mutation, overlapping copy, absent keys, closed channels, recursive payloads,
explicit defaults, invalid foreign tuples, retained callbacks, and typed nil errors.
Reject the design if it adds runtime cost or loses required Go behavior.
The proposal states a requirement; no compiler or benchmark result proves it yet.

## Sources

- [Go values](https://go.dev/ref/spec#Representation_of_values): shared reference data.
- [Slice rules](https://go.dev/ref/spec#Slice_expressions): bounds can use capacity.
- [Map reads](https://go.dev/ref/spec#Index_expressions): missing keys return zeros.
- [Clear](https://go.dev/ref/spec#Clear): slice elements are zeroed; map entries are deleted.
- [Channel close](https://go.dev/ref/spec#Close): closed receives can return zero values.
- [Comparison](https://go.dev/ref/spec#Comparison_operators): comparable types and interface limits.
- [Go slices](https://go.dev/blog/slices-intro): backing arrays, length, capacity, and aliases.
- [Typed nil errors](https://go.dev/doc/faq#nil_error): interface nil differs from pointer nil.
- [Race detector](https://go.dev/doc/articles/race_detector): Go synchronization requirements.
- [Package types](https://pkg.go.dev/golang.org/x/tools/go/packages): inspect actual Go signatures.

These sources describe Go. Static initialization, defaults, and enum rules are the tgo proposal.
