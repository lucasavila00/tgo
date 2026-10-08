# Rewrite rules and review cases

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

## Which Go zero values are valid?

- Scalars: valid.
- Pointers, slices, maps, channels, functions, and interfaces: nil is valid.
- Structs: valid only if every field has a valid zero.
- Arrays: valid only if every element has a valid zero. A zero-length array is valid.
- Enums: invalid. Tag zero is reserved.

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
A generic Go function may still produce an invalid foreign value; checked reads handle that case.
Named result slots need explicit assignment before any read or bare return, for all types.
Their signature declarations do not count as initialization.

## Collection operations

For element types whose zero is invalid:

- `make([]T, 0, cap)` reserves storage; no zero element is readable.
- `append` adds supplied valid values.
- Reslicing may not increase the current length. Known violations are compile errors.
  Dynamic bounds use a runtime length check. Shrinking and restoring length is also rejected.
- `clear(slice)` is a compile error. It would create invalid elements.
- Element writes, `copy`, iteration, and shortening remain available.

For zero-valid element types, keep Go allocation, reslicing, and clearing rules.
Keep bounds panics and nil behavior. No automatic conversion of a nil map to an empty map.
Map assignment requires valid keys and values. `delete` and `clear(map)` remove entries.

Bulk operations must not bypass checks. For `copy(dst, src)`, check the selected source prefix
of length min(len(dst), len(src)), then use ordinary Go copy. For `append(dst, src...)`, check
all appended elements. Checks inspect tags and inline fields, not referenced graphs.
Checks execute no user callbacks. Keep overlapping-slice behavior. Concurrent mutation needs
caller synchronization, as it does in Go.

## Missing values

A map miss, closed-channel receive, or failed assertion can return an invalid zero.
For such result types, require a checked binding:

```text
if account, ok := accounts[id]; ok {
    use(account)
} else {
    // account is not in scope here.
}
```

The compiler binds the model value only in the successful branch. It checks the loaded value
there: a foreign map can contain an invalid enum even when the key exists.
Changing an unrelated bool cannot make an unchecked value usable.
Plain reads are allowed when the result zero is valid. Channel range stops at close,
and checks each received model value. Nil channels retain their blocking behavior.

## Enum storage and equality

Use a tag plus separate typed Go payload fields. Do not overlay memory with unsafe unions.
Go's garbage collector must see references in every payload field.
Only the active payload is a source value. Constructors zero inactive storage.
Inactive references must not remain after variant replacement.
This layout can use more space than a Rust enum. Stack allocation is not guaranteed.

References break recursive size dependencies. `Children []Node` and `Next *Node` are valid.
An enum directly containing itself by value has infinite size and is rejected.
Check inline struct and array payloads recursively. Do not follow references during this check.
Reference cycles therefore do not require a whole-graph traversal.

Equality compares tag, then active payload. All payloads must be comparable.
A slice or map field makes the enum non-comparable, even if that variant is inactive.
Interface fields keep Go's dynamic comparability checks and possible panics.
Require zero inactive storage at construction and checked reads. Reject foreign values
with nonzero inactive fields. This does not interpret inactive fields as source values.
For comparable enums, raw Go equality then agrees with tag-and-active-payload equality.
Use the generated comparable Go struct as the map key; no custom hash table is needed.

## Full Go FFI

Use native Go references. Preserve aliases, overlapping slices, interior pointers, pointer keys,
cycles, nil, callbacks, and object identity. Do not silently serialize or deep-copy graphs.
Import named Go types with their identity. Bind functions, methods, interfaces, channels,
variadic calls, and generic calls from Go type information. No scalar-only admission list.
Go runtime objects, including synchronization objects, retain their Go rules.

Export model type names with private representation fields. Go must be able to name these
types in signatures, fields, containers, and callbacks. Zero values remain possible.
Generate checked constructors, functions, and variant accessors. Carry source model metadata
with variant and accessor names. Other generated packages use the accessors, not private fields.
The checks enforce validity. Type-name privacy never prevented inferred generic zeros.

Copy and check every enum when source code reads it. This includes ordinary variables,
inline model fields, dereferences, indexing, map reads, channel receives, assertions,
callback arguments/results, and outgoing function arguments. Check captured locals again
if a Go alias could have changed them. The compiler may remove a check only with proof.
A copy of an enum can still contain references; check those values when later read.

Validate direct enum inputs before an exported source body runs. Apply the same rule to
constructor inputs and inline model fields in structs and arrays. Check valid enum outputs
before export. Opaque interfaces are not traversed; asserting a model type performs its check.
Errors keep their exact Go values, including typed nil and wrapping chains.

An invalid read panics with a generated `ffi.InvalidValue` value. Go can recover that panic.
Do not add a result parameter, replace an existing error, or recover unrelated panics.
This preserves interface method signatures and callback types.
A delayed check may fail after earlier effects. There is no transaction or rollback.

## Foreign result slots

Go commonly returns a zero value with an error. Capturing a foreign tuple preserves its slots;
it does not claim that each slot is a valid source model value.
Checking the error or discarding another slot does not read that slot.

```text
a, err := legacy.LoadAccount()
if err != nil { return err }
use(a) // Load, copy, and check here.
```

Every source use loads and checks a model slot, including assignment, argument passing,
storing into a container, matching, and returning it. The initial foreign tuple binding is
the sole capture exception. Taking its address must retain the checked-read rule.
A valid partial value remains usable on error. A nil error does not prove validity.
An invalid value fails when read, regardless of the separate error result.

A source function that needs an absent result can return a pointer with nil, or an explicit
result enum. Do not invent a zero model value for an error path.
Exact unchecked forwarding can stay in Go adapters. It uses Go rules and does not produce
a trusted source value. Do not silently skip checks to preserve a foreign zero enum.

## Aliases and callbacks

A Go caller can create zero with a literal or a generic function. Private fields do not prevent it.
A Go function can retain a pointer or slice and overwrite its enum storage after validation.
Checks must run on subsequent reads, not only on initial entry.

```go
func Wipe[T any](items []T) { clear(items) }
```

Calling Wipe with an enum slice retains the same storage. Later source reads fail.
Calling it with an int slice yields zeros that remain valid. The FFI does not block the Go call.

Callbacks keep the Go runtime lifetime. Check model arguments on entry and model results on use.
Retained callbacks and re-entry use the same rules. Channels check delivered values on receive.
Keep the caller's synchronization duties. Concurrent Go mutation without synchronization is a race;
validation cannot make it safe. Do not claim race freedom or safety from unsafe memory writes.

Go struct values do not record which fields were explicitly assigned. FFI cannot reconstruct
that history. Enforce source initialization syntax locally and runtime value rules at foreign reads.

## Evidence to compare

Use the same business task in Go and the proposed language. Count source and adapter code.
Compare review effort, extra checks, allocation, and payload storage costs.
Include shared mutation, overlapping copy, absent keys, closed channels, recursive payloads,
explicit defaults, invalid foreign tuples, retained callbacks, and typed nil errors.
Reject the design if necessary checks or lost Go behavior outweigh the clearer source rules.

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

These sources describe Go. Checked reads, explicit defaults, and enum rules are this proposal.
