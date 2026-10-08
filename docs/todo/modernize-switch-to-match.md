# ADR: Modernize eligible switches to match

Status: proposed

## Context

TGo has `match`, but it accepts only TGo enums. TGo code still uses Go switches for other
closed choices. This splits one idea between two forms and makes match less useful.

`internal/tgolint/modernize_error_return.tgo` has two examples. `staticErrorCallName` uses a
type switch on `ast.Expr`. `zeroConstant` uses a value switch on `constant.Kind`. Both functions
select one known form and return one result.

## Decision

Extend `match` with patterns for Go interface values and named Go constants. Keep the current
enum patterns and their exhaustive check.

A type pattern names one concrete Go type and binds its value:

```text
match expression {
case *ast.Ident(identifier): return identifier.Name, true
case *ast.SelectorExpr(selector): return selector.Sel.Name, true
}
return "", false
```

A constant pattern names one or more constants of the subject type:

```text
match value.Kind() {
case constant.Bool: return !constant.BoolVal(value)
case constant.String: return constant.StringVal(value) == ""
case constant.Int, constant.Float, constant.Complex:
    return constant.Compare(value, token.EQL, constant.MakeInt64(0))
}
return false
```

A match on an open Go type can omit cases and continue after the match. `_` is an optional
default case. A match on a TGo enum stays exhaustive.

The compiler emits a Go type switch for interface patterns and a Go value switch for constant
patterns. It evaluates the subject once. It keeps branch order, branch scopes, labels, `break`,
`continue`, and `goto` behavior. It adds no wrapper, reflection, allocation, or extra branch.

## Diagnostic

Add a `tgolint` modernization diagnostic for handwritten `.tgo` files. Report a switch only
after the compiler accepts the equivalent match syntax.

A type switch is eligible when all these rules are true:

- The switch has no init statement.
- Its subject has an interface type.
- Each non-default clause has one non-nil concrete type.
- The switch variable is absent or can become one binding in each type pattern.
- The clauses have no behavior that match cannot preserve.

A value switch is eligible when all these rules are true:

- The switch has no init statement and has one subject expression.
- The subject has a named basic type.
- Each non-default case is a constant assignable to that exact named type.
- Each constant case can become one match case without a new conversion.
- No clause uses `fallthrough`.

The diagnostic gives the match form at the switch position. It does not report generated Go,
ordinary `.go` files, tagless switches, `nil` cases, interface type cases, mixed constant types,
dynamic case expressions, or switches that need changed control flow.

The check uses the typed syntax tree that `tgolint` already has. It visits each switch and its
case clauses once. It resolves types and constants by object identity. It needs no call graph,
package search, or data-flow pass.

## Runtime requirement

The generated Go must have the same switch shape as the input form. Benchmarks must show no new
allocation and no material change in time for interface and constant matches. If direct lowering
is not possible for a form, the compiler must reject that form and the diagnostic must stay
silent.

## Scope

This decision adds two match pattern forms and one modernization diagnostic. It does not make Go
interfaces closed. It does not infer all possible implementations or constants. It does not
change TGo enum exhaustiveness.
