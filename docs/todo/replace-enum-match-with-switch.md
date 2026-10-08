# ADR: Replace enum match with tag switches

Status: proposed

## Context

TGo has two ways to read an enum. TGo source uses `match`. Generated Go and Go callers use an
ordinary switch on `TgoTag` and call the payload accessor for the active tag.

```text
match account {
case Personal(person): return person.Name
case Business(business): return business.Company
}
```

The compiler lowers this statement to a Go switch. The generated switch has a temporary for the
subject, one numeric case for each variant, one payload accessor call in each case, and a default
panic for an invalid foreign value.

Thus, `match` does not give a different runtime model. It gives a short source form and these
compile-time services:

- It evaluates any subject expression once.
- It binds the selected payload.
- It requires each variant exactly once.
- It adds the invalid-tag panic.
- It rejects `fallthrough`.

These services require a separate grammar production, public syntax nodes, parser markers,
lowering code, nested-match type-check passes, and match-specific diagnostics. `match` accepts
only TGo enums. A prior proposal to add Go type and constant patterns would make this separate
control form much larger. The current language does not have those pattern forms.

The repository already checks the equivalent Go form. `tgolint` requires an exhaustive
`TgoTag` switch before it permits a payload accessor. It also tracks writes, aliases, captures,
loops, `goto`, deferred calls, and goroutines that can invalidate the tag proof. This check is
the same kind of contextual proof that TGo uses after a nil or presence check.

## Decision

Remove `match`. Permit generated enum methods in TGo source only in this ordinary Go form:

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

The compiler must keep this switch as a Go switch. It must not add a wrapper, a hidden payload,
reflection, or a second dispatch. Enum construction and representation generation stay the same.

The case values are the existing numeric tags. Tag zero is invalid. Known tags start at one in
variant declaration order. This decision does not generate tag constants and does not add a new
variant-pattern syntax.

For a subject that is not already a stable local value, use a switch initializer:

```text
switch account := loadAccount(); account.TgoTag() {
case 1:
    person := account.TgoPersonal()
    use(person)
case 2:
    business := account.TgoBusiness()
    use(business)
default:
    panic("invalid Account variant")
}
```

This form evaluates `loadAccount` once and gives all cases the same receiver.

## Contextual checks

The compiler must apply the enum switch check during every TGo build. `tgolint` must apply the
same rule to Go callers. A payload read is valid only when all these statements are true:

- The switch tag is a direct `TgoTag()` call on a concrete enum value or its type alias.
- The receiver is an unaliased local value. It is not a pointer, field, package variable,
  interface value, or open type parameter.
- Every known numeric tag has one case. Go already rejects a repeated constant case.
- Each known-tag case has one tag. Cases cannot combine tags.
- The switch has a default path that cannot continue after the switch.
- No case uses `fallthrough`.
- A `TgoVariant()` payload call uses the same receiver as the tag call.
- The payload call occurs only in the case for that variant tag.
- The receiver does not change, escape, or become mutable between the tag call and payload call.
- The proof does not enter a function literal, deferred call, or goroutine.

Assignment to the receiver ends the proof. Taking its address, capturing it in a closure, or
calling a pointer-receiver method also ends the proof when that operation can change the value.
A loop or backward `goto` cannot carry the proof across a possible change. A new exhaustive
switch can establish a new proof.

These rules follow the existing `tgolint` enum switch analysis. The compiler and `tgolint` must
accept and reject the same source shapes. The implementation can share the analysis or test both
implementations with the same cases.

The default path normally panics because Go code can construct an invalid enum through its zero
value. A return or another path that cannot continue after the switch also satisfies the proof.
The default must not fall through to code that assumes a known variant.

## Exhaustiveness and payload safety

Exhaustiveness is based on the enum model, not only on the integer type of `TgoTag`. A switch is
incomplete when it omits a declared variant, even when it has a default case. A case outside the
known range is an error. A non-constant case is an error.

The check ties each payload accessor to one variant number. For the example, `TgoPersonal()` is
valid only in case 1, and `TgoBusiness()` is valid only in case 2. A payload accessor remains
invalid outside the checked case. A tag call also remains invalid outside an exhaustive switch.

This is a compile-time rule. Accessors keep their current generated code and do not add a runtime
tag check. Boxed payload access keeps its current type assertion. The terminating default handles
tag zero and unknown foreign tags before any payload read.

## Migration

Replace each match mechanically:

1. If the match subject is not a stable local value, bind it in the switch initializer.
2. Replace each variant case with its one-based declaration index.
3. Add an explicit payload accessor assignment when the old binding was not `_`.
4. Add a default that panics or otherwise cannot continue after the switch.

For example:

```text
match account {
case Personal(person): use(person)
case Business(_): useBusiness()
}
```

becomes:

```text
switch account.TgoTag() {
case 1:
    person := account.TgoPersonal()
    use(person)
case 2:
    useBusiness()
default:
    panic("invalid Account variant")
}
```

The migration must update the specification, agent guidance, repository TGo source, generated Go,
compiler tests, and `pkg/syntax` users. Remove `MatchStmt`, `MatchCase`, `MatchStatement`, and the
match query and traversal API from `pkg/syntax`. Remove parser-marker and match-lowering code after
all source files use switches.

The compiler must reject `match` after the migration. It must not keep a second compatibility
path. This is a source-language and syntax-API break. It does not change enum layout, constructors,
payload types, generated methods, or the Go API. Existing Go tag switches stay valid.

## Runtime behavior

The switch has the same dispatch, payload copy, and invalid-tag path that `match` emits now. The
change must add no allocation and no extra payload copy. Generated code tests must compare the
migrated switch shape with the old match output. Allocation tests must cover inline and boxed
payloads.

## Drawbacks

Numeric cases are less clear than variant names. A reader must compare each number with variant
declaration order or with the payload accessor in that case. Inserting or reordering variants can
require edits to every switch.

The source form is longer. Users must write the payload assignment and invalid-tag default.
Compiler diagnostics can identify the variant name for a missing or wrong numeric case, but they
cannot make the case label self-documenting.

Named contextual case labels, such as `case Account.Personal`, would remove the numeric-label
problem. They would also need new source recognition and lowering because the label is not a Go
constant. This decision defers that syntax. It keeps the current generated API and uses the
ordinary switch that the repository already checks.

## Scope

This decision removes enum `match` and permits contextually checked `TgoTag` and payload accessor
calls in TGo source. It does not add patterns for interfaces, constants, nested payload fields, or
guards. It does not add generated tag constants, runtime tag checks, or a general flow-sensitive
pattern system.
