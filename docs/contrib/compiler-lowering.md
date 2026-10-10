# Compiler lowering

The compiler must preserve valid TGo behavior before it improves generated Go.
Use this order for each compiler change:

1. Preserve valid TGo semantics.
2. Preserve evaluation order, scope, control flow, and error identity.
3. Add fresh locals, blocks, or other Go structures when they are necessary.
4. Make the generated Go stable and readable.

Reject source only when the TGo program is invalid or supported Go cannot
represent its semantics. Do not reject valid source because a lowering is
difficult. Do not replace a feasible compiler lowering with a linter rule. Do
not require a user to write a temporary variable or other compiler
bookkeeping.

## Lower through a typed function plan

Compiler lowering must use this phase order:

1. Check the source and collect its semantic facts.
2. Build one typed, structured lowering plan for each function.
3. Emit Go AST from that plan.

The plan builder must return typed value IDs or place IDs for each expression.
It must keep the source facts that later phases need, including captured source
types, contextual expected types, and required result counts. It must not
change the source AST or create generated Go statements.

The plan must give guarded, repeated, and selected work explicit regions. For
example, branches own guarded regions, loops own test and body regions, and
select cases own selected-case regions. The plan must also keep source scope,
object identity, and jump-target identity. These facts must not be recovered
from generated names or Go AST shapes.

All expression forms must use one recursive plan-building path. New expression
forms must extend that path. Statement lowering must define the execution
region for each expression, but it must not implement a separate expression
lowering path.

An assignment that needs lowering must have one assignment plan. Its builder
must plan recursive place preparation, all right-hand evaluation, and ordered
stores as separate regions. Place captures keep their natural source types;
underlying or normalized type-set shapes only select the storage policy.
Contextual constants can remain at their consumer. Other values must keep
their actual types and required result counts. A generic array-or-reference
place must use explicit planned alternatives. The emitter can distribute the
same prepared load or store recipe through those alternatives, but it must not
choose captures, evaluation order, or store timing.

Only the emitter can turn plan values into Go identifiers and plan operations
into Go statements. The emitter must consume the plan directly. It must not
call the replaced expression lowerer, use stale generated-AST maps, or recover
types or identities from generated names.

## Preserve semantics

Evaluate each source operand at the same point and the same number of times as
Go. A failure path must skip all later work that the source would skip. Keep
short-circuit evaluation and conditional case evaluation.

Keep each source name in its normal lexical scope. Use a generated block when
lowered statements must stay with a statement initializer. A generated local
must use the compiler fresh-name service. It must not capture or be captured by
a source name.

Preserve each `break`, `continue`, `goto`, label, and `fallthrough` target. A
loop rewrite must keep the post statement on every path where Go runs it.

For `!!`, return the same error interface value. For `!`, wrap the same error
one time with the specified call name. Keep normal deferred calls on every
generated return path.

Generated Go must pass the formatter after semantic tests pass. Readable output
is important, but it cannot change source behavior.

## Review a lowering

Use focused tests for these properties when they apply:

- success, failure, and skipped side effects;
- left-to-right evaluation and one-time evaluation;
- lexical scope and source-name collisions;
- unlabeled and labeled control flow;
- nested functions and deferred calls;
- exact `!` wrapping and `!!` error identity; and
- formatted, deterministic generated output.

Use run-time tests when an output comparison cannot prove behavior. A linter
rule can enforce language safety. It cannot hide a missing compiler lowering.

Diagnostics must distinguish invalid TGo from an unimplemented lowering. Use
`compiler does not yet lower` for a confirmed compiler gap. Link the gap to a
focused issue with a regression example.

## Valid lowering contexts

Error propagation and comprehensions use the same recursive expression plan
at each expression evaluation point in a function. This includes loop
initializers, conditions, and post statements; switch tags and cases; type
switch statements; select communications; assignments; and nested calls.

Direct propagation on a `go` or `defer` call remains invalid because that call
executes outside the current function return point. Package-scope propagation
and comprehensions remain invalid because no function body exists for their
generated control flow.
