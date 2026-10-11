# Transpiler architecture

The transpiler reads a Go module with `.tgo` files and emits ordinary Go.
It uses Go's scanner, parser, type checker, and build selection. It does not
modify the Go toolchain. See the [user guide](../../guide/transpiler.md) for
commands and the public API.

## Package dependencies

| Package | Compiler packages it calls |
| --- | --- |
| `cmd/tgo` | `pkg/transpiler` |
| `pkg/transpiler` | `internal/driver`, `internal/lowering` |
| `internal/driver` | `internal/frontend`, `internal/lowering` |
| `internal/frontend` | None |
| `internal/lowering` | None |

`cmd/tgo` reads arguments and writes files. `pkg/transpiler` exposes module
compilation and expression-statement insertion. The driver loads modules and
runs the compiler passes. The frontend prepares postfix operators for Go
parsing. Lowering changes typed expressions into statements. Frontend and
lowering do not import each other.

## Prepare and check the source

For this source:

```text
value := strconv.Atoi(text)!!
```

The frontend replaces `!!` with a unique comment after the call. Go can parse
the remaining source. The compiler records the operator's position and whether
it must wrap the error.

The driver finds calls whose result types are available. It checks that each
call ends with an error result, then replaces the marked call with a generated
helper call. The helper has the call's success results but excludes its error.
For `Atoi`, its result is `int`. This lets Go check the surrounding assignment.
Nested propagation needs repeated loading because an inner helper can make
an outer call's type available.

The fully prepared AST receives Go type checking. Two further checks reject
propagation inside `unsafe.Sizeof`, `Alignof`, or `Offsetof`, and error results
that the containing function cannot return. A helper's success-only signature
cannot check the return statement that lowering will add.

## Replace helpers with calls and error checks

The helper is for type checking. It does not handle errors at runtime. Lowering
replaces it with the original call, typed saved results, a failure check, and
an ordinary Go return.

For example, the essential statements for `consume(target()!!)` are:

```go
var tgo1 int
var tgo2 error
tgo1, tgo2 = target()
if tgo2 != nil {
    return tgo2
}
consume(tgo1)
```

The complete function also has return storage. On failure, `!!` returns the
original error and zero values for preceding results. `!` wraps the error once
with the source call name and `%w`. Deferred calls still run.

The failure check precedes conversion into the caller's error result. Thus a
nil pointer error is success, but an error interface that contains a nil pointer
is failure. Generic results follow the instantiated type. Reflection handles
cases where Go cannot compare the value directly with nil.

## Keep evaluation and scope

An expression rewrite returns statements and references to evaluated results.
Its parent runs the statements and then uses those references. For
`consume(first(), target()!!, last())`, lowering saves `first()` before calling
`target()`. If the error check returns, `last()` never runs.

Conditional work stays conditional. In `left() && right()!!`, the call to
`right()` stays inside the true branch. Loop conditions and post statements
stay on their required visits. Select operands run before selection; receive
assignment targets run in the chosen case.

Assignment lowering saves destination operands without reading the destination.
Array stores use the original array. Slice and map stores use the saved container.
Address checks and store checks keep their distinct execution points.

Named functions and function expressions have separate return targets, including
functions in package initializers. A label on a declaration stays on the first
added statement; later declarations remain in the source scope.

Generated variables have explicit checked types and fresh names. For a shift
used as `uint64`, `var saved uint64 = 1 << n` keeps that type; `saved := 1 << n`
would use `int`. Type printing uses type identity, not spelling. The compiler reports an error if Go cannot
name a required type in that scope. It does not fall back to an inferred type.
Changed syntax is copied so the public insertion API can reuse the typed input.
Every accepted propagation marker must be processed.

## Output and checks

`name.tgo` produces `name_tgo.go` beside its source. Test and platform suffixes
stay last: `name_linux_test.tgo` produces `name_tgo_linux_test.go`. Existing
output does not participate in source checking. Build constraints select source
files. Line directives affect diagnostic locations; physical paths control
file selection and output.

The [correctness report](correctness.md) explains the implementation checks and
independent behavior tests. The [test guide](../../../tests/readme.md) describes
fixtures, snapshots, and expression-insertion tests.
