# WHAT

## Decision

Check TGo source with a private copy of Go's type checker and a matching
compiled-package reader. Check propagation and value-producing expressions
together before lowering. Use Go's standard parser through the frontend wrapper.

This replaces the stock-checker requirement in the
[expression-lowering plan](../expression-lowering/how.md). It makes the checker
choice in the [value-expression plan](../value-expressions/how.md) apply to
module compilation, including `!` and `!!`.

## Reason

The current driver makes propagation calls readable by Go's type checker with
generic helpers. Each helper accepts all call results and returns only the
success results. The driver writes at most one helper per file in each round,
then loads the module again. Independent calls in one file therefore require
separate loads. Nested calls can require another round before the outer call
has a valid type.

A private checker can check `outer(inner()!)!` in one expression traversal.
It checks the inner call, validates propagation, gives that expression its
success result types, and continues with the outer call. It needs no intermediate
helper declaration or module reload for each operator.

Value-producing `if` expressions also need branch types, contextual types,
generic inference, scopes, and control flow checked together. Separate source
transforms would make those rules harder to keep consistent.

[PR #301](https://github.com/lucasavila00/tgo/pull/301) implements this direction.
This decision does not state that the PR's implementation is complete.

## Consequences

The lowerer still owns evaluation order and the Go statements that preserve
source behavior. Direct source checking does not replace lowering. Check the
emitted Go with the stock Go checker as a separate final step.

Keep public APIs and ordinary Go behavior. The public `Load` API retains stock
`packages.Package` and `go/types` objects. Private checking uses separate type
objects. Preserve the [automatic modernization rules](../../problem/README.md#adoption-and-automatic-modernization).

The compiler must maintain patches against a pinned Go source version. Go
updates require patch review and compiler validation. This cost is accepted
because source rules can then be checked directly without repeated helper-based
module checking. Runtime improvement must be measured; no timing claim is made.

Batching independent helpers could reduce loads for propagation alone. It would
retain checking helpers and would not supply the source rules needed by
value-producing branches. A patched parser alone would not supply those type
rules either.

[HOW](how.md) defines the stages. [PROOF](proof.md) defines the checks.
