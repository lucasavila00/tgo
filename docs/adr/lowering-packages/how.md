# Lowering packages: HOW

## Approach

Keep public Source, Site, After, and Propagation in lowering. Shared reads
source.Inventory directly. Propagation owns error checks, wrapping, and failure
returns. Comprehensions own collection construction, ranges, and output insertion.
Valueexpr owns if and switch expression results.

Shared owns typed values, evaluation, storage, assignments, ordinary Go control
flow, and request, file, and function state. Each value carries its expression,
checked type, source expression, and temporary status. Results carry statements
and values. Storage expands tuple values only when needed.

## Example

Operand scheduling for `fetch()!` can encounter an if expression. Shared calls
the dispatcher, which selects valueexpr. Recursive callbacks receive the current
shared context explicitly:

```go
type Expression func(*shared.Context, ast.Expr) shared.Result
type Statement func(*shared.Context, ast.Stmt) []ast.Stmt
```

A nested function gets new function state and retains file and request state.
Callbacks must not capture the parent function context. Rule calls use their
source function context, so returns and jumps retain their targets.

## Tradeoffs

Two callbacks break the recursive dispatch dependency. Do not add an interface
that repeats every rewrite method. Do not copy the pending APIs from #336 or
#346; move their merged contracts.
