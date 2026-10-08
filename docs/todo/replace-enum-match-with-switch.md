# ADR: Replace enum match with tag switches

Status: proposed

## Context

TGo `match` accepts only TGo enums. The compiler lowers it to an ordinary switch on `TgoTag`.
It evaluates the subject once, binds each payload, checks exhaustiveness, rejects `fallthrough`,
and adds a default panic.

This form has no different runtime behavior. It needs separate grammar, syntax nodes, parser
markers, lowering, and diagnostics. `tgolint` already checks the equivalent Go switch and permits
each payload accessor only when its tag case proves that the payload is active.

## Decision

Remove `match`. Permit generated enum methods in TGo source only in a checked Go switch:

```text
func Label(account Account) string {
    switch account.TgoTag() {
    case 1:
        person := account.TgoPersonal()
        return person.Name
    case 2:
        business := account.TgoBusiness()
        return business.Company
    default:
        panic("invalid Account variant")
    }
}
```

Tag zero is invalid. Variant tags start at one in declaration order. A switch initializer, such
as `switch account := loadAccount(); account.TgoTag() {`, can evaluate and bind a subject once.

The compiler keeps the source switch. It does not add another dispatch or runtime tag check.
This decision adds no tag constants or named case-label syntax.

## Contextual checks

The compiler applies these rules to TGo source. `tgolint` applies the same rules to Go source.

- The tag is a direct `TgoTag()` call on a concrete enum or its type alias.
- The receiver is a stable path rooted in an unaliased local value. It is not a pointer, package
  variable, interface value, or open type parameter.
- Each case value is a constant tag in the known range. Every known tag occurs exactly once.
- The switch has a default path that cannot continue after the switch.
- No case uses `fallthrough`.
- A payload accessor uses the same receiver as `TgoTag()` and occurs in a case that contains only
  its matching tag. A case can combine tags only when it does not read a payload.
- The receiver does not change or escape between the tag read and the payload read.
- The proof does not enter a function literal, deferred call, or goroutine.

Assignment, address-taking, capture, or a pointer-receiver call ends the proof when it can change
the receiver. A loop or backward `goto` cannot carry the proof across such a change. A new checked
switch can establish a new proof.

`TgoTag()` is invalid outside a checked switch. A payload accessor is invalid outside its proven
case. The generated accessors keep their current code and do not check the tag at runtime.

## Consequences and migration

This is a source-language and syntax-API break. Replace each match as follows:

1. Bind an unstable subject in the switch initializer.
2. Replace each variant case with its one-based declaration index.
3. Call the matching payload accessor when the old case used its binding.
4. Add a default path that cannot continue.

Remove the match grammar, parser markers, lowering, diagnostics, and the `MatchStatement` and
`MatchCase` syntax API. The compiler does not keep a compatibility path.

Enum layout, constructors, payload types, generated methods, and the Go API do not change.
Existing Go tag switches stay valid. A migrated switch has the same dispatch, payload copy, and
invalid-tag path as the old match. It adds no allocation.

Numeric cases are less clear than variant names. Inserting or reordering a variant can require
changes to every switch. Named contextual labels would need new lowering, so this decision defers
them.
