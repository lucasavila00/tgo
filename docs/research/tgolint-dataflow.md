# tgolint data-flow notes

The official
[`buildssa`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/buildssa) pass
builds SSA and a control-flow graph for each source function. The official
[`nilness`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/nilness) analyzer
uses SSA dominance for branch facts.

The current tgolint implementation does not use SSA dominance. It uses
[`go/cfg`](https://pkg.go.dev/golang.org/x/tools/go/cfg) to remove unreachable syntax.
It reads supported path conditions from the typed AST. It stores these conditions in
the exported generic effect fact. It marks other control paths as unresolved.

For a generic call, `will` means that the analyzed operation will run and its unsafe
event is proved. The analyzer uses these event rules:

- A variable, named result, `new`, or literal hole produces a zero value.
- `make` uses a positive slice length.
- `clear` operates on a slice that is proved to be nonempty.
- A map key is proved to be absent.
- A channel read returns its zero value when the channel is closed. The current check
  can prove that a fresh channel is open. Other closed states remain unresolved.
- A one-result type assertion does not create a zero value. It returns the existing
  asserted value or stops with a panic. The zero-effect check does not report it.
- A comma-ok assertion is proved to fail.
- A reslice is proved to extend the length within the capacity.
- A tgo accessor is on a path that is proved to run.

A fresh channel is proved to be open. A constant key in a map literal is proved to be
present or absent. A concrete assertion is proved to succeed or fail. Constant slice
bounds prove whether `clear` or a reslice produces a zero. These proofs can suppress a
diagnostic. A mutable condition parameter, a runtime value, and an unsupported control
path remain unresolved. The diagnostic uses `can` for these cases.

A local generic function or method value is checked at each direct call. The call
arguments decide its value conditions. An unresolved escape reports that the effects
cannot be checked at a call site. Creating a returned closure does not run its body.
The exported fact records effects in a direct returned function literal, and tgolint
checks these effects when the returned function runs.

The returned-function fact does not follow a closure through an intermediate local
or another helper. Such code can hide an effect from the current checker. Keep the
function literal in the return statement until returned-function flow is complete.

The constructor and presence-pair branch check uses the CFG from the branch target.
It reports a branch only when a path can use the pending value before a matching proof
or a new assignment kills the value. This check includes `break`, `continue`, `goto`,
and `fallthrough`.

## Boundary validation contract

An `error` result does not validate a foreign enum. A function can return an `Event`
with an invalid tag and a nil error. The current generated API has tag and payload
accessors, but it has no validation operation that tgolint can identify as a complete
runtime check. A successful type assertion can also import an existing invalid tgo
value. It is an ingress validation problem, not zero construction.

A checkable contract needs a generated operation such as:

```go
func ValidateEvent(value Event) (Event, error)
```

The operation must reject unknown tags, read only the payload for the active tag,
validate each nested model, and return a value rebuilt with the matching constructor.
Checked nested values must run their predicates again. Application nil and ownership
rules still belong to the boundary module because the type declaration does not state
them.

The compiler must generate this operation and export an analysis fact that identifies
it before tgolint can treat a nil error as proof of a valid foreign enum. This change
does not add that public API.
