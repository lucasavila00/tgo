# Add an explicit successful return

## Problem

Postfix `!` and `!!` remove repeated failure branches. Success still repeats `return value, nil`.

The syntax must be explicit so ordinary Go source stays unchanged. It must also lower without a
type query or result-count inference.

## Decision

Add the contextual statement keyword `succeed`:

```text
SuccessStmt = "succeed" [ ExpressionList ] .
```

It appends one untyped `nil` to an explicit return list:

```text
succeed              -> return nil
succeed value        -> return value, nil
succeed left, right  -> return left, right, nil
```

The example becomes:

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!
	succeed user.Name
}
```

Use `succeed` with no expressions for a function that returns only an error. Named results do not
change the rule. List each successful named result: `succeed result`. Generic expressions and
error aliases need no special handling because Go checks the lowered return.

Calls with one success result work: `succeed repo.Find(id)!` or `succeed repo.Find(id)!!`. A direct
multi-value call does not combine with the added `nil` in Go. Bind its values first:

```go
left, right := Pair()
succeed left, right
```

The statement means "append nil," not "infer an error." It can therefore type-check when the final
result is another type that accepts `nil`, exactly as an explicit Go `return ..., nil` can.

## Parsing and lowering

The parser gives an ordinary Go statement form precedence. Thus, `succeed(value)` is a call, and
`succeed <- value` is a channel send. Labels, selectors, declarations, and assignments also keep
their Go meaning. Bind an intended parenthesized first value or leading receive before `succeed`.
No valid Go program contains the remaining statement form, so ordinary Go stays byte-identical.

The parser records the statement and its source span. Lowering replaces the keyword with `return`
and inserts `nil` after the expression list. It adds no temporary, helper, branch, type query, or
allocation. Existing `!` and `!!` lowering then processes their calls in the normal order.

Explicit expressions keep their positions. Generated `return` uses the keyword position, and
generated `nil` uses the statement end. Syntax diagnostics point to `succeed`; Go result-count and
assignment diagnostics point to the original statement or expression.

## Alternatives

PR #51 proposes `return value,`. That form is shorter and has the same code-generation cost, but
its trailing comma is easy to miss and Go can later give that punctuation a meaning. `succeed` is
a verb like `return`, states the success path, and does not suggest generator behavior as `yield`
does. `success` is a noun and reads less clearly as a statement.
