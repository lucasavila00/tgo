# Short returns for values and errors

Status: proposed.

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

## WHAT

Add two forms of `return`. The function's final result must implement Go's
built-in `error` interface. The earlier results are its success results.

| TGo | Meaning in a function returning `(int, string, error)` |
| --- | --- |
| `return .. err` | `return 0, "", err` |
| `return count, name ..` | `return count, name, nil` |

A leading `..` supplies the zero value of each success result's declared type.
It requires one error expression, assignable to the final result type. A trailing
`..` supplies nil for the final result. Explicit values must fill every success
result and must fit their declared types under Go assignment rules.

Use these forms for the issue's examples. Do not add `Ok` or `Err` functions:
a return statement already has the result types needed to supply missing values.
A library function would need explicit type arguments for omitted result types
and separate forms for different result counts.

```text
func Parse(text string) (int, error) {
    value, err := strconv.Atoi(text)
    if err != nil {
        return .. err
    }
    return value ..
}
```

Both forms return unconditionally. `return .. err` does not test, wrap, or
replace `err`. If `err` is nil, the function returns zero success values and nil.
If a typed nil pointer converts to a non-nil error interface, retain that result.

Accept named error interfaces, concrete error types, and type parameters whose
constraints require `error`. The trailing form additionally requires nil to be
assignable to the final type. Thus a pointer error can use both forms; a struct
error can use only the leading form. A generic final type can use the trailing
form only when nil is assignable to every type in its constraint.

In a function returning only an error, `return .. err` means `return err`.
A bare `return ..` is incomplete: the leading form needs an error operand, and
the trailing form needs success operands. Use the existing `return nil` for
error-only success. Do not infer omitted operands from named result variables.

A single expression that produces several values can fill the success results:
`return pair() ..` is valid when `pair()` supplies exactly those results. Do not
mix a multiple-result expression with other values. Existing TGo expressions
are valid operands, including `return load()! ..`. Their returns, defers, and
jumps retain the enclosing source function and control-flow targets.

Named results follow explicit Go return behavior. Evaluate operands first, then
assign all result variables, then execute deferred calls. A leading `..` replaces
previous success values with zero values; a trailing `..` replaces a previous
error with nil. For example, this function returns `1, nil`:

```text
func Count() (count int, err error) {
    count = 9
    defer func() { count++ }()
    return .. nil
}
```

## HOW

Read `..` as a return operator, with its two dots adjacent. Keep Go selectors,
floating-point literals, and `...` unchanged. Follow Go's token boundaries:
write `return 1 ..`, since `1..` starts with the floating-point token `1.`.
The operator is valid only at the start or end of a return's operand list.
It cannot fill a middle result or appear on both sides of the same list.

Apply Go's semicolon rules before the operator and within operands. After a
trailing `..`, insert a semicolon at a newline or end of file. A leading `..`
still needs its operand, which can start on the next line. Comments follow
these same rules. Do not interpret dots inside comments or strings as operators.

The [frontend](../../internal/frontend/readme.md) must record the operator,
its operands, and their original positions on a return statement. Prepare a
Go-parsable return with a compiler marker, then let the existing parser read its
operands. A marker is an internal syntax node, with no runtime function call.
Keep it attached to the return so that checking does not depend on a user name.

The [checker](../../internal/checker/readme.md) must use the enclosing source
function's result types. Check the leading operand against the final result;
check trailing operands against the success results. Supply these types to
constants, nil, if values, and generic inference, as for ordinary returns.
Count declared results, including grouped names and blank result names.
Treat either form as a return for missing-return and branch-completion checks.
Real function literals have their own result types. The placeholders used to
parse if values and comprehensions do not introduce a source function.

Report malformed syntax at `..`. Report incompatible operands at their original
`.tgo` positions, with the expected type or result count. Use the built-in error
type by identity, even when source code declares a name `error`.

The [lowerer](../../internal/lowering/readme.md) must emit an ordinary explicit
return. Reuse the typed zero storage used by error propagation for omitted
success values. These fresh locals must hold zero values independently of named
results. For example:

```text
func Fail[T any](err error) (T, error) {
    return .. err
}
```

has the behavior of:

```go
func Fail[T any](err error) (T, error) {
    var zero T
    return zero, err
}
```

For trailing `..`, append nil after lowering the supplied values. If the source
shadows `nil`, use fresh zero storage of the final result type. Resolve generated
types by their checked identity, so local type names cannot change their meaning.

A sole multiple-result call needs locals before the final return:

```go
first, second := pair()
return first, second, nil
```

Use fresh names and the checked types when a conversion needs context. Evaluate
each operand once and preserve the order Go defines. Use the existing expression
lowering for branches, propagation, and saved earlier operands. A return inside
an operand can exit before the outer return completes. Do not add a helper
function around operand evaluation: that would change the targets of returns
and the time at which deferred calls run. Keep generated declarations inside
blocks where necessary to preserve valid jumps. The final generated source
must pass the stock Go checker.

Follow the [modernization rules](../problem/README.md#adoption-and-automatic-modernization).
An automatic fix can replace explicit returns only when it proves the same
result values and effects. For a leading form, each removed operand must be a
zero value of its destination type, with no effects. A variable named `zero`
is not sufficient evidence. A zero concrete value converted to an interface
can be non-nil and must not become the interface's zero value. For a trailing
form, the omitted expression must produce the same nil result without effects.
An empty non-nil slice, a typed nil inside an error interface, and a call that
returns nil do not meet these rules. Leave uncertain cases unchanged.

## PROOF

Add `.tgo` fixtures, generated Go snapshots, and independent Go reference
programs under `tests/testdata/`. Compile and run both forms against the same
expected results and effect traces. A trace records calls and deferred actions
in execution order. References must use explicit Go returns, without compiler
lowering code.

Check these cases:

- The three issue examples, error-only returns, multiple success results, and
  a sole multiple-result call. Confirm each call executes once.
- Zero values for numbers, strings, arrays, structs, pointers, slices, maps,
  functions, channels, interfaces, named types, and generic success results.
- Named results that already hold values, grouped and blank result names,
  shadowed result names, and defers that read or change those results.
- Nil errors, concrete pointer errors, struct errors in the leading form,
  richer error interfaces, generic error constraints, and interface conversion
  of typed nil pointers. Compare error identity as well as `errors.Is` and
  `errors.As` behavior.
- Contextual constants and shifts, nil success values, generic inference, and
  if values. Include nested functions with different result lists.
- Operand calls with effects, propagation success and failure, operand returns,
  panics, defers, and jumps. Confirm skipped operands stay skipped and outer
  result assignment does not occur when an operand exits first.
- Comments, multiline operands, selectors, decimal literals, `...`, and
  ordinary returns in the same file. Check original source positions.
- Source names that collide with generated locals or shadow `nil`, `error`,
  and result types. Snapshots must retain the intended type identities.

Require diagnostics for missing or extra success values, incompatible operands,
a final type that does not implement error, nil incompatible with the final
type, multiple error operands, both operator positions together, and use of
`..` outside a return. Check termination analysis in if values and functions
whose only exits use these forms.

For automatic fixes, compare execution before and after each accepted change.
Require unchanged source for calls with effects, nonzero values, non-nil empty
collections, typed nil errors converted to interfaces, other conversions that are not zero, and
variables whose zero value cannot be proved. Include safe constants and typed
zero declarations as accepted cases.
