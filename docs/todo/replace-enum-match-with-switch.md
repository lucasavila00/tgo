# ADR: Replace enum match with tag switches

Status: proposed

## Context

TGo `match` accepts only TGo enums and lowers to `switch value.TgoTag()`. It adds a separate
grammar, syntax API, and lowering pass for behavior that a Go switch already provides. `tgolint`
already uses the tag switch as contextual proof for payload access in Go source.

The generated enum API has `TgoTag()` and one payload accessor per variant, such as
`TgoPersonal()`. Tags start at one in declaration order. Tag zero is invalid.

## Decision

Remove `match`. Permit generated enum methods in TGo source in this canonical form:

```text
func Label(account Account) string {
    switch account.TgoTag() {
    case 1:
        return account.TgoPersonal().Name
    case 2:
        return account.TgoBusiness().Company
    default:
        panic("invalid Account variant")
    }
}
```

Keep numeric cases. Do not generate tag constants. Numeric tags are already part of the TGo API,
and constants would add public names without changing dispatch or payload safety. Named case labels
can be a separate decision if numeric cases become too costly to maintain.

Payload accessors do not check the tag and add no allocation. Their wrong-variant behavior follows
the current storage:

- An inline accessor returns its inline payload slot. For a value made by another variant
  constructor, this slot has its zero value.
- A boxed accessor asserts the dynamic payload type. A wrong variant causes that assertion to
  panic.
- An empty-payload accessor returns its empty payload value for any tag.

The contextual check prevents these wrong-variant calls in checked source. It does not add a
runtime guard to an accessor.

## Contextual checks

The compiler applies these rules to TGo source. `tgolint` applies the same rules to Go source.

- The proof starts at a direct `switch value.TgoTag()` on a stable enum receiver. The receiver is
  a path rooted in an unaliased local value. A pointer, package variable, interface value, or open
  type parameter does not establish the proof.
- Each case value is a constant tag in the known range. Every known tag occurs exactly once, no
  case uses `fallthrough`, and the default path cannot continue after the switch.
- A case with one tag permits only that variant payload accessor on the same receiver.
- A case with multiple tags proves no payload and permits no payload accessor.
- Assignment to the receiver invalidates the proof for later payload reads.
- A nested closure does not inherit the proof. It needs its own checked tag switch.
- `TgoTag()` is invalid outside a checked switch. A payload accessor is invalid outside its
  proven case.

A switch initializer can evaluate an expression once and bind the stable receiver:
`switch value := load(); value.TgoTag() {`.

## Consequences and migration

Replace each match with a tag switch, use the one-based declaration index for each case, call the
payload accessor when the old case used its binding, and add a terminating default. Remove the
match grammar, parser markers, lowering, diagnostics, and the `MatchStatement` and `MatchCase`
syntax API. Do not keep a compatibility path.

This is a source-language and syntax-API break. Enum layout, constructors, payload types,
generated methods, and the Go API do not change. A migrated switch has the same dispatch and
payload copy as the old match. It adds no allocation.

Numeric cases are less clear than variant names. Inserting or reordering a variant can require
changes to each switch.
