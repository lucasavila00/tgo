# Switch expressions produce one value

Status: proposed. Issue: [#263](https://github.com/lucasavila00/tgo/issues/263).

## WHAT

Use Go switch syntax to select a value. This extends
[if expressions](value-expressions.md) without adding general block expressions.
Statements can precede a branch's final value:

```text
message := switch code {
case 200:
    audit(code)
    "ok"
default:
    "error"
}
```

Support Go initializers, switches without a tag, and type switches. The tag is
the expression after `switch`. Evaluate it once. Evaluate cases in source order
and use the first match. Use `default` only if no case matches, regardless of
its position. A switch without a tag compares its cases with `true`.

```text
size := switch value := input.(type) {
case string:
    len(value)
case []byte:
    len(value)
default:
    0
}
```

Each path that reaches a branch end must supply one value. A return, panic, or
jump out needs none. Returns, defers, and error propagation belong to the source
function. Reject `fallthrough` between value branches and `break` targeting the
value switch. Keep valid jumps to outer constructs and breaks inside nested
loops or ordinary switches. Ordinary switch statements retain Go behavior.

Require `default`, except when `exhaustive:` proves that cases cover every form
of an enum. An enum declares a closed set of forms. Check coverage against the
resolved declaration, not names that resemble enum members. General expression
switches and type switches still need `default`.
[Enum support](../problem/vision.md#list-every-allowed-form) is a dependency for
this exception; it is not implemented in the current compiler.

Use the if-expression type rules: context can supply the type; otherwise use
matching typed branches and Go constant rules. Nil alone supplies no type.
Completing branches participate together in generic inference. The result is
one nonconstant value. If every branch exits, require context or a conversion
to establish its type.

## HOW

Use one typed result variable and ordinary Go control flow. For the first
example, the basic output inside a function is:

```go
var result string
switch code {
case 200:
    audit(code)
    result = "ok"
default:
    result = "error"
}
message := result
```

A generated function call is a simpler expression form, but it changes the
meaning of source `return`, `defer`, and jumps. Direct statements preserve these
operations in their source function. Generated variable names must be fresh.

Extend the frontend's conditional records to carry expression switches and
type switches. Preserve original positions. Go parsing still reads headers and
clauses. Parsing placeholders must not create source function boundaries.
The private checker checks each clause in its original scope. Initializer
bindings cover the switch; type-switch bindings retain Go's clause-specific
types. Record result types, completing paths, and jump targets before lowering.

Replace each completing final expression with an assignment to result storage.
Reuse [expression lowering](expression-lowering.md) to place the switch where
its value executes. Save earlier operands and preserve skipped operands,
assignment order, repeated loop evaluation, and package initialization.
Use the existing package-initializer rules when no source function encloses it.

Retain a Go switch when its case expressions need no added statements. Otherwise,
save the tag once and check cases in order while no case has matched. Save the
selected clause number, then dispatch to its body with an ordinary switch.
Only select the default clause after all cases fail. Keep type switches as Go
type switches so bindings retain their types. Enum coverage uses enum declaration
information; this decision does not define a new enum representation or boundary
check.

## PROOF

Extend the [value-expression fixtures](../../tests/readme.md) with switch input
files, generated Go snapshots, expected output, and independent handwritten Go
references. Compile and run both implementations. Compare values and event logs;
snapshot agreement alone does not prove behavior.

Check these observable results:

- An initializer runs before the tag. The tag runs once. Cases run left to right,
  including multiple expressions in one clause. A match skips later cases.
  A default written first runs only after all cases fail.
- An expression switch without a tag selects the first true case. Type switches
  preserve single-type, multiple-type, nil, and default bindings and scopes.
- Nested if and switch values preserve calls, short-circuit operands, loop
  visits, assignment destinations, and package initialization order.
- A source return skips later work. Defers run at function exit. Outer jumps
  reach their original targets; nested loop breaks remain valid. Panics keep
  their value and order. `!` returns the same error; `!!` adds call context and
  preserves `errors.Is` and `errors.As` access to the cause.
- Contextual types, named types, nil, constants, and generic calls follow if
  rules. Arithmetic on the result preserves runtime overflow.
- Complete enum coverage permits `exhaustive:`. Missing forms, duplicate cases,
  and a same-spelled member from another declaration cannot prove coverage.

Require source-position diagnostics for missing default or coverage, missing
values, incompatible types, multiple values, constant-only uses, escaped clause
bindings, value-switch breaks, and fallthrough between value branches. Include
valid ordinary switches with break and fallthrough to detect regressions.
