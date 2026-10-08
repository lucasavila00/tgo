# ADR: Modernize manual error returns

Status: proposed

## Context

TGo postfix `!` can replace one common Go error check. The replacement is safe only when the
manual check has the exact behavior that `!` generates. A broad style rule can change error text,
return values, or control flow.

## Decision

Add one modernization diagnostic to `tgolint`. Run it only on handwritten `.tgo` files. Do not
check `.go` files or generated files.

Report a manual error return only when these statements are next to each other:

```go
account, err := repo.Find(id)
if err != nil {
	return nil, fmt.Errorf("repo.Find: %w", err)
}
```

The TGo replacement is:

```text
account := repo.Find(id)!
```

The rule also covers a call that returns only `error`:

```go
err := store.Flush()
if err != nil {
	return fmt.Errorf("store.Flush: %w", err)
}
```

```text
store.Flush()!
```

## Exact pattern

Require all these AST and type facts:

- A short declaration has one call on its right side.
- The call results end in the predeclared `error` type.
- The last new variable receives that error result.
- The next statement is `if err != nil` with no initializer or `else`.
- The `if` body contains only one `return` statement.
- Each earlier return expression is the zero value of its result type.
- The last return expression resolves to the standard `fmt.Errorf` function.
- `fmt.Errorf` has one constant format and the same error variable as its only value argument.
- The format is an accepted static call name followed by `: %w`.
- The enclosing function has an unnamed final result of the predeclared `error` type.

An identifier call accepts only its name. A selector call accepts its full selector path or its
last selector. Thus, `load(id)` accepts `load`. `repo.Find(id)` accepts `repo.Find` and `Find`.
Type arguments do not change the accepted names.
Accept a zero only when its AST and type prove the value. This includes `nil` for a nilable type,
`false`, numeric zero, an empty string, and an empty literal of the exact result type.

## Exclusions

Do not report a check that adds context other than the exact call name. Do not report `%v`, a
different format, extra format arguments, another wrapper, or error translation.

Do not report cleanup, logging, metrics, assignments, deferred work, or any other statement in
the error branch. Do not report a changed success value, a nonzero error result, a named result,
or a branch that uses `else`, `goto`, `break`, `continue`, `panic`, or another return path.

Do not report a call through a dynamic function value. Do not report an assignment to existing
variables. These forms need more analysis and can have different scope or assignment behavior.

## Consequences

The diagnostic identifies a mechanical TGo edit. It does not offer a fix when any behavior can
change. Manual error handling remains valid when it adds useful work or context.
