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

## Current lowering audit

The audit at main commit
`aaed15ba0de180700b8ba2155ecedb2a68ada237` inspected compiler and linter
diagnostics. It reproduced each restriction below with valid TGo source.

| Context | Error propagation | Comprehension |
| --- | --- | --- |
| Type-switch initializer or assignment | [#195] | [#218] |
| `for` initializer | [#213] | [#218] |
| `for` post statement | [#214] | [#218] |
| Expression-switch case | [#215] | [#218] |
| Select communication | [#216] | [#218] |

The linter has no rule that rejects these contexts. The compiler owns the
listed diagnostics. Direct propagation on a `go` or `defer` call remains
invalid because that call executes outside the current function return point.
Package-scope propagation and comprehensions remain invalid because no
function body exists for their generated control flow.

[#195]: https://github.com/lucasavila00/tgo/issues/195
[#213]: https://github.com/lucasavila00/tgo/issues/213
[#214]: https://github.com/lucasavila00/tgo/issues/214
[#215]: https://github.com/lucasavila00/tgo/issues/215
[#216]: https://github.com/lucasavila00/tgo/issues/216
[#218]: https://github.com/lucasavila00/tgo/issues/218
