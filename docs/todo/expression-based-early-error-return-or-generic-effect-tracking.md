# ADR: Add automatic error propagation

Status: proposed

## Context

Go calls often return `(value, error)`. Most callers only add the operation name and return the
error. The repeated check hides the business code.

```go
account, err := repo.Find(id)
if err != nil {
	return "", fmt.Errorf("repo.Find: %w", err)
}
```

The operation names form a useful call chain. Today, each caller must write them by hand.

## Decision

Add postfix `!` to calls whose last result is the Go `error` type.

```text
account := repo.Find(id)!
```

The call still has the native Go `(value, error)` result. On success, `!` yields the values
before `error`. On failure, it wraps the error with the static call name and returns from the
current function.

The current function must also have `error` as its last result. The compiler returns zero values
for its earlier results.

## Call names

The compiler gets the name from the source call target. A selector uses its short source form. A
named function value uses its variable name.

```text
record := repo.Find(id)!
data := file.ReadAll()!
value := decode(data)!
```

These calls add `repo.Find`, `file.ReadAll`, and `decode`. If the compiler cannot get a short
static name, it reports an error at `!`. The caller can then use a normal Go error check.

Each `!` adds one name. A failure through two layers can produce this text:

```text
service.Load: repo.Find: sql: no rows
```

This is a small call trace for the error path. It has operation names, not program counters.

## Generated Go

This tgo code:

```text
func LoadLabel(repo Repo, id ID) (string, error) {
	account := repo.Find(id)!
	return account.Label, nil
}
```

lowers to normal Go:

```go
func LoadLabel(repo Repo, id ID) (string, error) {
	value, err := repo.Find(id)
	if err != nil {
		var zero string
		return zero, fmt.Errorf("repo.Find: %w", err)
	}
	account := value
	return account.Label, nil
}
```

The compiler creates fresh temporary names. It adds a fresh `fmt` import name when needed.
Wrapping happens only on the error path. The success path has the same call, error test, and
branch as a manual Go check.

The wrapper uses `%w`. `errors.Is`, `errors.As`, and `errors.Unwrap` keep normal Go behavior.

## Result shapes

`!` accepts a call with zero, one, or many values before the final `error`. An error-only call
can be an expression statement. Many success values can be used by a matching assignment.

```text
store.Flush()!
key, value := index.Entry(id)!
```

In a nested expression, the call must have one success value. The lowerer saves earlier operands
and keeps Go evaluation order. It does not evaluate later operands after an error.

Short-circuit expressions keep their conditional paths. Loop conditions check on each iteration.
Deferred call arguments run before registration, as Go requires.

## Custom errors

The function that creates an error sets its type, data, and message. Most callers use `!` and add
only their call name.

A caller can use a normal Go error check when it must classify an error, add runtime data, recover,
or return a different value. This explicit branch replaces `!` for that call.

## Compatibility

Generated functions keep their Go signatures. Go callers receive the normal `error` interface.
Go tools can read the generated checks and wrapped causes.

The source checker verifies both tuples before lowering. It reports errors at `!`. Existing Go
code keeps its current behavior because `!` is new tgo syntax.

This proposal adds no effect system. Failure stays visible in the function signature and at each
propagating call.
