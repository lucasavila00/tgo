# Short returns for values and errors

Status: proposed.

Issue: [#321](https://github.com/lucasavila00/tgo/issues/321)

## WHAT

Add two forms of `return`. The function's final result must implement Go's
built-in `error` interface. The earlier results are its success results.

| TGo | Explicit return |
| --- | --- |
| `return .. err` | `return 0, err` for `(int, error)` |
| `return value ..` | `return value, nil` |
| `return count, name ..` | `return count, name, nil` |

A leading `..` supplies each success result's typed zero value. It requires
one error operand, assignable to the final result type. A trailing `..` supplies
nil. Its explicit operands must fill every success result under Go assignment
rules. Both forms return unconditionally, without testing or wrapping errors.
A nil error stays nil; a typed nil pointer converted to a non-nil error interface
stays non-nil.

Accept named error interfaces, concrete errors, and type parameters constrained
by `error`. The trailing form also requires nil to be assignable to the final
type. Thus a pointer error permits both forms; a struct error permits only the
leading form. For a generic final type, nil must fit every allowed type.

In an error-only function, `return .. err` means `return err`. Bare `return ..`
is incomplete. Use `return nil` for error-only success.

A sole multiple-result expression can fill the success results: `return pair() ..`.
Do not mix it with other operands. Accept existing TGo expressions, including
`return load()! ..`. Keep their enclosing function and control-flow targets.

Named results follow explicit Go returns: evaluate operands, assign result
variables, then run defers. Omitted success results replace previous values
with zeros; an omitted error replaces the previous error with nil. This example
returns `1, nil`:

```text
func Count() (count int, err error) {
    count = 9
    defer func() { count++ }()
    return .. nil
}
```

Choose return syntax instead of `Ok` and `Err` helpers. The function signature
already supplies the omitted types and result count. Library helpers would
need extra type arguments and separate forms for different result counts.

## HOW

Read adjacent `..` dots only at the start or end of a return's operand list.
Reject a middle operator or both positions together. Preserve Go token
boundaries: write `return 1 ..`, since `1..` starts with the float token `1.`.
Keep selectors, `...`, strings, and comments unchanged.

Apply Go semicolon rules before the operator and within operands. Trailing `..`
ends a statement at a newline or end of file. Leading `..` needs its operand,
which can start on the next line. Comments follow these same rules.

The [frontend](../../internal/frontend/readme.md) must record the operator and
original positions on a return statement. Prepare a Go-parsable return with a
marker: an internal syntax node with no runtime call. Let Go's parser read
the operands.

The [checker](../../internal/checker/readme.md) must check operands against the
enclosing function's corresponding result types, including contextual constants,
nil, if values, and generic inference. Treat the statement as a return for
termination checks. Report syntax errors at `..` and operand errors at their
original `.tgo` positions, with the expected type or count.

The [lowerer](../../internal/lowering/readme.md) must emit an explicit Go return.
Reuse error propagation's fresh typed zero locals for omitted success values.
For `return .. err` in a function returning `(T, error)`, emit:

```go
var zero T
return zero, err
```

For trailing `..`, append nil. A sole multiple-result call needs fresh locals:

```go
first, second := pair()
return first, second, nil
```

Use checked type identities and fresh names. If source code shadows `nil`,
use zero storage of the final result type. Resolve built-in `error` by identity.

Use existing expression lowering to evaluate operands once, preserve Go's
required order, and save earlier values before later operations. Retain source
returns, defers, and jumps through generated blocks. A helper around operands
would change those effects. Generated output must pass the stock Go checker.

Follow the [modernization rules](../problem/README.md#adoption-and-automatic-modernization).
Replace explicit returns only when removed operands have no effects and equal
the omitted destination values. A variable named `zero` proves nothing.
Conversions matter: a zero concrete value inside an interface can be non-nil.
Leave uncertain cases unchanged.

## PROOF

Add `.tgo` fixtures, generated Go snapshots, and independent Go references under
`tests/testdata/`. Run generated and reference programs against the same expected
values and traces of calls and defers. References must use explicit Go returns.

Check:

- The three issue examples, error-only returns, sole multiple-result calls,
  every Go zero-value kind, named types, and generic results.
- Named results with previous values, grouped and blank names, shadowed names,
  and defers that read or change results.
- Nil, concrete and generic errors, richer interfaces, and typed nil pointers
  converted to interfaces. Check error identity, `errors.Is`, and `errors.As`.
- Contextual constants, shifts, nil, generic inference, if values, and nested
  functions with different result lists.
- Effects, propagation, operand returns, panics, defers, and jumps. Each reached
  operand runs once; an early exit skips later operands and result assignment.
- Comments, multiline syntax, floats, selectors, `...`, ordinary returns,
  original positions, and shadowed `nil`, `error`, and type names.

Require diagnostics for wrong result counts, incompatible types, missing
operands, invalid operator positions, and `..` outside returns. Check functions
and if-value branches whose only exits use these forms.

For automatic fixes, compare execution before and after accepted changes.
Accept proved typed zeros and nil. Leave effectful calls, non-nil collections,
nonzero interface conversions, and unproved variable values unchanged.
