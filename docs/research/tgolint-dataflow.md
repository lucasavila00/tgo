# tgolint data-flow notes

The official
[`buildssa`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/buildssa) pass
builds SSA and a control-flow graph for each source function. The official
[`nilness`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/nilness) analyzer
uses SSA dominance for branch facts.
The Go compiler also stores immediate dominators and uses a
[`SparseTree`](https://go.dev/src/cmd/compile/internal/ssa/sparsetree.go) for fast
dominance queries. Go analysis facts support separate analysis across package boundaries.

The current tgolint implementation uses
[`go/cfg`](https://pkg.go.dev/golang.org/x/tools/go/cfg) for local control flow. A forward
fixed-point analysis tracks Boolean and integer constants and direct parameter aliases.
Each join keeps only facts that agree on every incoming path. A proven constant branch removes
its dead edge. Typed AST conditions keep call parameters in exported generic effect facts.
Go analysis facts carry a separate wire form across packages. The checker decodes that form to
TGo sum types before use.

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
diagnostic. Scalar assignments and local aliases can make a call prove `will` or `never`.
Passing a Boolean or integer by value does not erase its fact. Taking its address or capturing
it in a closure does. Package variables, free variables, narrowing conversions, mutable
containers, runtime values, and unsupported paths remain unresolved. The diagnostic uses
`can` for these cases.

Model facts include the package path and type name. Two packages can declare the same type
name without merging their models. Generic unions that mix those models stay mixed.

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

An `error` result does not validate a foreign enum. A function can return an `Event` with an
invalid tag and a nil error. A successful type assertion also imports an existing value. It
does not validate the value.

The compiler now generates this contract for each tgo model:

```go
func ValidateEvent(value Event) (Event, error)
```

The operation rejects unknown tags, reads only the active payload, validates nested models,
and returns a value rebuilt with the matching constructor. Checked values run their predicates
again. A module-local generated runtime carries graph identity through imported validators.
It preserves repeated pointers, maps, and identical slice headers, and stops recursive cycles.
Overlapping slice views with different headers rebuild independently.

Generated methods reconstruct private model fields with typed assignments. Reflection is used
only during an explicit validation call to traverse ordinary Go containers. Each ordinary
private field is rejected because the validator cannot inspect it. A channel or function is shared only
when its static signature cannot transport a model. Interfaces in those signatures are rejected.

`tgolint` exports facts for the generated operation. It also exports a fact for a direct wrapper
that returns a generated validator or checked constructor call. A successful error proof for an
arbitrary `(T, error)` source does not mark its value valid. A successful proof for a generated
validator result does.

This analysis uses typed AST control flow and the existing pair-state merge. It does not use SSA
dominance for boundary taint. The current proof follows exact direct model parameters, pair
bindings, supported branches, direct forwarding wrappers, and simple local function values.
It does not prove cgo or `unsafe` memory, application ownership, or race freedom.
