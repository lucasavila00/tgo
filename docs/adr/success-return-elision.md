# Elide the final nil error on success

## Problem

Postfix `!` and `!!` remove the repeated failure branch. The success return still repeats `, nil`
in each function that returns a value and an error.

```go
func LoadName(repo Repo, id ID) (string, error) {
	user := repo.Find(id)!
	return user.Name
}
```

## Decision

A return can omit exactly one final result when that result is the predeclared `error` type. An
alias of `error` qualifies. A new defined error type and a type parameter constrained by `error`
do not qualify.

The return must have one or more expressions. They must supply all results before the final error
under Go assignment rules. This rule applies to functions, methods, and function literals. It
also applies to named results and generic earlier results.

```go
func Keep[T any](value T) (T, error) { return value }
func Named(value string) (result string, err error) { return value }
```

A bare `return` keeps its Go meaning. A function that returns only `error` must use `return nil`.
The rule does not omit two or more results.

A single multi-value expression can supply all earlier results:

```go
func Pair() (int, string)
func LoadPair() (int, string, error) { return Pair() }
```

The compiler evaluates `Pair` once, stores its results in fresh values, and returns those values
followed by `nil`.

## Resolution and lowering

The compiler first tests the return against the complete result tuple. If it is valid Go, the
compiler does not change it. Thus, `return load()` stays unchanged when `load` already returns
every result, including its error.

If the complete test fails, the compiler tests the expressions against the result tuple without
its final error. It uses the existing package `types.Info`, Go assignability, and inferred generic
instances. It does not add a checker or a type inference system.

The pass records eligible source returns after the first package type check. Existing expression
lowering then runs. The pass appends a generated `nil` after scalar results. For one multi-value
expression, it emits one direct assignment to fresh values before the return. This adds no
closure, helper call, or allocation.

Propagation lowering runs before the final return is emitted. Thus, `return load()!` uses the same
rule, and its generated failure branch keeps its wrapped error. The normal post-lowering package
check validates the emitted Go.

Explicit expressions keep their source positions. Generated values and `nil` use the return
statement position. Diagnostics do not report a synthetic file or line.

## Diagnostics and compatibility

This change adds no token and does not change a valid Go return. If the final result is not
`error`, the result count differs by more than one, or an earlier value is not assignable, the
compiler reports the normal Go return diagnostic at the original statement.
