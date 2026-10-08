# ADR: Add collection comprehensions

Status: proposed

## Context

[Python][python] and [Julia][julia] combine one result with generators and filters. [C#][csharp]
uses the same parts as query clauses. TGo needs the operation with Go types and control syntax.

## Decision

Permit one `for range` block inside a slice or map literal. Nested ranges add generators. Go `if`
blocks add filters. The deepest block contains one result.

```text
names := []string{for _, account := range accounts {
    if account.Active { account.Name }
}}
```

The literal gives the output type. Each range must use `:=`. Its variables exist only in its
block. A block contains one nested range, one filter, or one result. Use `&&` to combine filters.
All bindings, sources, conditions, and results are Go syntax.

```text
pairs := []Pair{for _, customer := range customers {
    for _, order := range customer.Orders { Pair{customer.ID, order.ID} }
}}
```

```text
byID := map[ID]Account{for _, account := range accounts {
    if account.Active { account.ID: account }
}}
```

An equal map key replaces its earlier value. Map range order stays unspecified.

## Lowering

The slice example lowers to ordinary Go in the surrounding function:

```go
__tgo_result := make([]string, 0)
for _, account := range accounts {
	if account.Active {
		__tgo_result = append(__tgo_result, account.Name)
	}
}
names := __tgo_result
```

The map example lowers to:

```go
__tgo_result := make(map[ID]Account)
for _, account := range accounts {
	if account.Active {
		__tgo_result[account.ID] = account
	}
}
byID := __tgo_result
```

The compiler uses private temporary names. Each source runs when its containing loop reaches it.
Each accepted result runs once. An empty slice result is non-nil.

Postfix `!` can occur in a source, condition, key, value, or slice result. It returns from the
surrounding function. Completed side effects remain.

## Scope

This change adds eager slice and map output. It adds no reducer, grouping, sorting, lazy result,
parallel work, `yield`, or collector protocol. Use a Go loop when the body needs statements,
mutation, `break`, `continue`, or more than one result path.

[python]: https://docs.python.org/3/reference/expressions.html#list-displays
[julia]: https://docs.julialang.org/en/v1/manual/arrays/#Comprehensions
[csharp]: https://learn.microsoft.com/dotnet/csharp/linq/get-started/query-expression-basics
