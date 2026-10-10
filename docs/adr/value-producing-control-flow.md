# Add value-producing control flow

## Context

Go `if` and `switch` constructs are statements. Code that selects a value must
repeat an assignment or return in each branch. TGo needs a small expression
model that removes this repetition without making every block an expression.

## Decision

TGo will allow `if` and `switch` in an expression position. Their branch bodies
will use the final expression as the branch value.

```text
label := if account.Active {
    account.Name
} else {
    "inactive"
}
```

An `if` expression must have an `else` branch. Each path that reaches the end of
a branch must supply one value.

```text
message := switch code {
case 200:
    audit(code)
    "ok"
default:
    "error"
}
```

A `switch` expression must have a `default` clause or prove that its cases are
exhaustive. An existing enum `exhaustive:` clause can provide that proof. A
value-producing switch cannot use `fallthrough` or a `break` that exits the
switch.

All branch values must have one result type. The compiler will use Go
assignment and untyped constant rules to check the values. The expression will
produce one value in the first version. It will not produce a Go multi-value
result.

The constructs will use the existing Go headers for `if` and `switch`. A normal
statement position will keep normal Go behavior. TGo will not change ordinary
blocks into expressions.

`return`, `defer`, and error propagation inside a branch will keep their
meaning in the enclosing function. The compiler will lower the expression to a
temporary and ordinary Go control flow. It will not use an immediately invoked
function because a function boundary would change these operations.

Lowering will preserve Go operand evaluation order when the expression is
nested in a call, operator, literal, or other expression.

## Consequences

- Existing Go and TGo source keeps its meaning.
- The parser, syntax tree, formatter, type checker, and lowering pass must
  support the two expression forms.
- Generated Go will contain a temporary and a statement form of the control
  flow.
- Pattern matching can later use the same branch-value rules.
- General scoped block expressions remain unavailable.

## Alternatives

A general `do {}` expression would allow any statement block to produce a
value. It adds syntax and control-flow cases that the initial use cases do not
need.

Making every block an expression would be a larger departure from Go. It would
also make existing statement contexts and semicolon behavior harder to explain.
