# ADR: Modernize enum-shaped iota

Status: proposed

## Context

Go permits patterns that TGo can express with stronger types. A named integer type with
`iota` can model a closed set, but any integer conversion can create an unknown value. It also
cannot require each switch to handle every member. A TGo enum closes the set and lets
`tgolint` check each match.

The TGo compiler must continue to accept Go syntax. Modernization is a source check, not a
language restriction.

## Decision

Add one modernization diagnostic to `tgolint`. Run it only on handwritten `.tgo` files.
Do not check `.go` files. A diagnostic fails `tgolint` in the same way as its safety diagnostics,
but it does not make the syntax invalid.

The first rule reports an enum-shaped `iota` declaration. A declaration has this shape when:

- one `const` group declares two or more values of the same defined integer type;
- `iota`, including an implicit repeated expression, gives the values; and
- the values are unique and sequential after one common offset.

```go
type State uint8

const (
	StateReady State = iota
	StateRunning
	StateStopped
)
```

Report this message at `iota`:

```text
State is an iota enum; use a TGo enum to close its variants
```

The TGo replacement is explicit:

```text
type State enum {
    Ready struct{}
    Running struct{}
    Stopped struct{}
}
```

Do not report bit sets. Their values can be combined, so they are not closed alternatives.

```go
type Permission uint8

const (
	PermissionRead Permission = 1 << iota
	PermissionWrite
)
```

Do not add an automatic fix. The integer values can be part of a public or stored format. A
conversion to a TGo enum needs an explicit boundary mapping.

## Rule scope

This ADR adds only the enum-shaped `iota` rule. A later modernization rule must name one safer
TGo form, define an exact source pattern, and exclude Go forms with different semantics. Do not
add a general list of banned Go syntax.

The rule has no runtime effect. It adds no generated code, wrapper, or validation. It uses the
same package loading and source positions as the current `tgolint` checks.

## Consequences

A file starts this check when its extension changes from `.go` to `.tgo`. The developer then
gets a direct prompt to replace an open integer state set with a closed TGo enum.

A sequential integer set can still be a protocol code or stored value. The rule reports it
because its source shape is the same as an enum. The developer must keep an integer boundary
mapping when those numeric values are part of an external contract.
