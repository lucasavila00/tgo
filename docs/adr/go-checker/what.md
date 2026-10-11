# WHAT

Check handwritten Go that uses generated TGo types. Prevent it from creating
or exposing those values through default initialization or bypassing their
construction rules. Ordinary Go types retain ordinary Go zero-value behavior.

Every named type declared in TGo carries initialization and construction
requirements into Go, including ordinary structs and named scalar types.
Identify a generated TGo type by its package and declaration. Follow aliases,
embedded fields, nested structs and arrays, collections, pointers, interfaces,
and generic type arguments. Equal names in different packages are different
types.

Generated type rules state which values are valid and how construction proves
validity. Explicit scalar zero remains valid when supplied under those rules;
a default declaration does not supply a value. Enforce enum variants, checked-type validation, and non-null fields
where the language defines them. A Go literal, conversion, field write, or
new defined type must not bypass those rules.

Check every operation that can supply a forbidden TGo default: declarations,
named results, omitted aggregate fields, allocation, clearing, reslicing, map
misses, closed receives, failed assertions, and failed calls. Check indirect
calls, callbacks, aliases, and generic functions too.

Reserved storage can be filled before use if the analysis proves every required
component valid and no forbidden value is exposed. A failed result can travel
with its error or presence status, but cannot be consumed before success.
Accept every construction and control-flow pattern the analysis can prove.

Analyze reachable dependencies and attribute failures to uses in the selected
Go package, with the dependency origin. Do not require standard-library changes
or reject an entire dependency when its safe uses can be proved.

Report possible forbidden zeros and construction violations with source
position, type identity, origin, path, and generic condition. Unknown effects
remain possible violations when they can produce or expose a protected value.
`unsafe` and cgo can break the guarantee; record these boundaries. Do not add
runtime checks.

Track implementation in [issue #267](https://github.com/lucasavila00/tgo/issues/267).

The [TGo checker](https://github.com/lucasavila00/tgo/issues/266) applies initialization rules to all TGo
values. This checker applies generated TGo type rules at the Go boundary.
[HOW](how.md) defines the implementation; [PROOF](proof.md) defines acceptance.
