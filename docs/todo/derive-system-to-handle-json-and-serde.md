# ADR: Add a staged JSON derive system

Status: proposed

## Context

tgo models use private fields and constructors to keep invalid values out of tgo code.
`encoding/json` cannot decode these models safely by itself. An enum becomes `{}`. A checked
type also becomes `{}`. A tgo struct can receive an invalid zero in a missing nested field.

The current boundary rule says that code must decode a transport value, validate it, and then
construct a model value. Repeated handwritten adapters are safe, but they are easy to omit.

## Decision

Add one top-level derive declaration:

```text
derive JSON(Message)
```

The grammar is `DeriveDecl = "derive" "JSON" "(" TypeName ")" .`.

`derive` and `JSON` are contextual in this declaration. The target must be a local tgo struct,
enum, or checked type. It must be the declared type name, not an alias. A duplicate derive is an
error. A source declaration that conflicts with a generated name is also an error.

The compiler emits `MarshalJSON` and `UnmarshalJSON` methods. Thus the type implements the
standard Go JSON interfaces. There is no tgo codec interface and no codec registry.

## JSON semantics

A checked value uses the JSON form of its base value. Decode the base value and call `NewT`.
Assign the receiver only after the constructor succeeds. Encode must run the predicate again.
This check rejects an invalid checked value that came from Go.

A tgo struct uses a JSON object. Exported field names and `json` tags follow `encoding/json`.
Unexported fields do not occur in JSON. A missing field uses its declared tgo default. If it has
no default, it gets Go zero only when that zero is valid. Otherwise, decode returns an error.

The first version rejects embedded fields. It rejects an ignored field with an invalid zero
and no default. It rejects an omit tag for any field with an invalid zero. Unknown and duplicate
fields keep the `encoding/json` rules.

An enum uses an externally tagged object with exactly one known variant:

```json
{"Personal":{"Name":"Lucas"}}
```

The variant name is the JSON key. The value is the variant payload object. Decode rejects an
empty object, more than one distinct key, and an unknown key. It decodes the payload and calls
the generated variant constructor. Encode switches on the tag and rejects tag zero or an
unknown tag.

Each statically reachable model in an encoded field must also derive JSON. The compiler follows
named fields, pointers, and collection elements. For an array with an invalid-zero element,
decode rejects input with too few elements. Nil pointers and collections keep their valid zero.
Dynamic interface values stay a trust boundary because their type is not known at compile time.

Every decoder uses a temporary result. It does not change the receiver after any error.

## Generated Go

For a checked type, decode has this form. Encode validates with `NewQuantity` before it calls
`json.Marshal` on the base value.

```go
func (v *Quantity) UnmarshalJSON(data []byte) error {
	var raw int
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	next, err := NewQuantity(raw)
	if err != nil {
		return err
	}
	*v = next
	return nil
}
```

Enum methods use a generated switch and a `json.RawMessage` payload. Tgo struct methods use a
generated wire struct plus presence flags where defaults or invalid zeros require them. The
compiler adds `encoding/json` with a fresh import name, as it does for other inserted names.

Generated methods call only `encoding/json`, generated constructors, and direct field code.
The compiler emits no tables or registration calls. Derive changes no model layout and adds no
cost to construction, access, matching, or calls that do not use JSON. JSON work has the same
operations as a safe handwritten boundary adapter.

## Safety and compatibility

Decode always reconstructs checked and enum values. It never writes private representation
fields. Nested invalid-zero fields must be present, have a default, or fail. This keeps the
current zero-validity and foreign-value rules.

Go callers use `json.Marshal` and `json.Unmarshal` without an adapter. Existing code changes
only when a type adds the derive. The wire form is then a public contract. Variant renames,
field tag changes, and checked base changes can break stored data.

## Drawbacks

The enum wire form is a tgo choice and can differ from an existing service contract. The first
version does not support embedded tgo struct fields. Generated struct decoders are larger than a
plain `json.Unmarshal` call. Predicate checks during encode can expose invalid Go input that old
handwritten code did not reject.

## Staged decision

1. Add the derive parser, name checks, and checked-type JSON methods.
2. Add enums with the exact externally tagged form and invalid-tag tests.
3. Add tgo structs, defaults, nested models, arrays, tags, and atomic receiver tests.
4. Update the specification only after all three model kinds pass Go and tgo boundary tests.
5. Add another derive name only when a second codec has a stable Go interface and wire rules.

Do not add generic Serde traits in these stages. Reuse the declaration and code-generation path
for a later codec. Each codec must still generate direct Go with no common runtime layer.
