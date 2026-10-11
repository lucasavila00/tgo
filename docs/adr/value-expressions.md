# If expressions produce one value

## Decision and reason

An if value executes one branch and uses its final expression. Require `else`
and use Go headers, including initializers. Branches can contain statements;
ordinary blocks do not become expressions. This permits a conditional value
without requiring a helper function that changes return or defer behavior.

Returns, defers, propagation, and jumps belong to the enclosing source function.
A path that cannot complete supplies no value or inference type. A `continue`
cannot target a range while its collection is being computed.

## Types and lowering

Assignments, returns, arguments, and conversions can supply the type. Check
completing branches with Go assignment rules. Otherwise, typed branches must
have the same type; constants use Go's constant-kind and default-type rules.
Nil alone supplies no type. Do not infer a new interface or convert unrelated
named types. Require context or conversion when all branches terminate.

Supply one nonconstant result. Reject multiple results and constant-only uses.
Array `len`, `cap`, and `range` evaluate if values; unsafe size and offset
operands retain their evaluation rules.

The [checker](../../internal/checker/readme.md) checks branches in their source
context. The [lowerer](../../internal/lowering/readme.md) emits result storage
and ordinary branches. Preserve operand order, package initialization, and
control-flow targets. Required generated types must be nameable. See the
[user guide](../guide/transpiler.md) for syntax examples.

## Validation

Use `tests/testdata/value-expressions/` snapshots and independent Go references.
Check selected and nested branches, initializers, locals, outer names, calls,
literals, assignments, skipped operands, returns, defers, labels, and jumps.
Check propagation identity, package initialization, comments, positions, and
fresh names.

Cover named types, interfaces, nil, constants, shifts, and generic inference
from several conditional and ordinary arguments. An `int8` context for
`(if ok {127} else {0}) + 1` must preserve runtime overflow.

Require diagnostics for missing values or else, incompatible types, tuples,
constant-only uses, escaped locals, unknown types, and inaccessible generated
types. Retain valid body and outer-loop jumps while rejecting continue into
an unfinished range collection. See [tests](../../tests/readme.md) and
[fuzzing](../../tests/fuzzing.md) for validation methods.
