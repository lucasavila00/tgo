# ADR: Add `%T` non-null pointers

Status: proposed.

## Decision

Add `%T` as a non-null pointer to `T`. Keep `*T` as a nullable Go pointer.
The grammar addition is `NonNullPointerType = "%" Type .`.

```text
type User struct {
    Manager *User
    Account %Account
}

func LoadAccount(id ID) (%Account, error)
```

`Manager` can be nil. `Account` is always non-nil. Both types lower to Go pointers.

The prefix operators compose at each pointer level:

```text
*T   nullable pointer to T
%T   non-null pointer to T
*%T  nullable pointer to a non-null pointer to T
%*T  non-null pointer to a nullable pointer to T
%%T  non-null pointer to a non-null pointer to T
```

Each `%` lowers to `*`. Pointer aliases keep the null rule of their target.

## Type rules

The zero value of `*T` is valid. The zero value of `%T` is invalid. Existing TGo rules for
explicit initialization, fields, arrays, collections, and named results apply to `%T`.

`&value` and `new(T)` produce `%T`. A `%T` value is assignable to `*T`. A nullable pointer is
assignable to `%T` on a path that proves the value is non-nil.

```text
account := legacy.LoadAccount(id)
if account == nil {
    return ErrMissingAccount
}
use(account) // account is %Account on this path
```

Assignment, address escape, capture, or a call that can change the pointer ends the proof.
A control-flow join keeps the proof when every incoming path proves a non-nil value.

Pointer indirection and implicit pointer field selection need `%T` or a local non-nil proof.
A method with a `*T` receiver keeps Go nil behavior. A method with a `%T` receiver requires a
non-null value.

Nested pointer types are invariant. For example, `[]%T` and `[]*T` are different TGo types
because a later write can change the collection contract.

## Go lowering

The compiler erases `%T` to `*T` in fields, parameters, results, aliases, nested types, and
function types. The generated Go API keeps pointer identity, method sets, and calling rules.

Generated package facts record each non-null pointer position. `tgolint` uses the facts to
check Go calls, returns, field writes, literals, and zero values.

A generated function with a `%T` parameter or receiver checks it for nil at entry. A failed
check panics and names the parameter or receiver. APIs that want a recoverable nil case use
`*T` and return an error when required.

## Foreign values

A `*T` value from Go remains nullable after an error check. TGo code proves it non-nil before
using it as `%T`. This rule covers callbacks, assertions, decoders, cgo, reflection, and
`unsafe`.

Generated validators reject nil in `%T` fields and nested `%T` positions. They preserve nil in
`*T` positions and then run the current reconstruction and model checks.

An interface keeps Go nil behavior. A non-nil interface can contain a typed nil pointer.
An assertion that produces `%T` checks both the dynamic type and the dynamic pointer value.

## Scope

This decision changes pointer types only. Slices, maps, channels, functions, and interfaces
keep their current Go nil rules. Existing `*T` declarations keep their meaning.

Implementation needs parser, formatter, type-checker, lowering, validator, fact, linter, and
flow-analysis changes. Acceptance needs tests for nesting, zero validity, flow proofs, Go
calls, entry checks, validators, package facts, and typed nil pointers.
