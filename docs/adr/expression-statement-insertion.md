# Insert returns through expression lowering

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Context

An expression can execute between other evaluations or only on a conditional
path. Moving an inserted statement before its enclosing statement changes
behavior. We need to insert `return err` before or after a selected evaluation
in ordinary Go. This establishes the algorithm needed by #247.

## Decision

Use one typed execution model: evaluation steps, conditional regions, and
explicit exit targets. This is expression normalization with control-flow
lowering. Keep source scopes and bindings attached to the model.

Separate the work into three phases:

1. Type-check unchanged Go input. Record source types, constants, bindings,
   result counts, assignment locations, and control-flow targets. Select the
   expression and insertion side before any rewrite.
2. Build the execution model from those facts. Each expression produces
   ordered steps and result references. Each statement places those steps in
   its execution region. Insert the return at the selected boundary.
3. Emit ordinary Go with fresh locals, blocks, branches, and labels. Preserve
   source declarations and scopes. Check the generated Go, then format it.

Split evaluations only where insertion requires it. Keep constants as
constants and preserve contextual conversions. Distinguish a computed value
from an assignment location; copying an array must not replace its location.
Do not depend on type information for an AST after changing that AST.

Keep conditional operands inside their branches. Separate assignment operand
preparation, RHS evaluation, and ordered stores. Route loop `continue` exits
through the post region. Preserve switch case order and fallthrough targets.
Keep select operand preparation before selection and receive target evaluation
inside the selected case. Preserve range evaluation rules and per-iteration
target preparation. Keep existing label and jump targets attached to source
statements; generated declarations must not invalidate a source jump.

The inserted return exits the source function and runs its deferred calls.
Nested functions have separate models. A function wrapper cannot implement
this return. For `go` and `defer`, arguments evaluate in the source function;
the scheduled call executes separately.

## Consequences

Keep the experiment separate from production and insert only `return err`.
No SSA, optimizer, or general inserted-statement API is required.

Prove every context in #248 with executable traces against handwritten Go.
Check both insertion sides, skipped paths, repeated execution, panic timing,
error identity, and deferred calls. Provide one focused test command.
