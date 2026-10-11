# Transpiler architecture

TGo programs become ordinary Go programs. The compiler must preserve their
execution while adding error propagation. Go supplies package selection and
ordinary type rules. The compiler owns the meaning of TGo operators and the
statements needed to implement them.

## Keep responsibilities separate

| Package | Responsibility | Reason |
| --- | --- | --- |
| `cmd/tgo` | Choose paths and write output | File-output policy belongs to the command. |
| `pkg/transpiler` | Expose compilation and expression rewriting | Other tools need the same compiler without command-line policy. |
| `internal/driver` | Select and check a module | Imports, build constraints, and tests must agree on package contents. |
| `internal/frontend` | Record added syntax | Go syntax keeps Go meaning; TGo operations retain their source identity. |
| `internal/checker` | Check TGo source types and scope | Added expressions need source rules before Go statements are generated. |
| `internal/lowering` | Emit equivalent Go statements | This requires checked types and the expression's execution context. |

Dependencies follow this order. Frontend and lowering do not call each other.
The driver supplies checked syntax and operation records to the lowerer.

## Preserve execution

Consider:

```text
consume(first(), target()!, last())
```

`first()` must run before `target()`. If `target()` fails, return its error
before `last()` or `consume()` runs. The lowerer saves earlier values and puts
the error check immediately after the call.

For `left() && right()!`, the right call and its check run only when `left()`
is true. Loop conditions keep their repeated execution. A receive assignment
keeps its destination evaluation inside the selected case. Assignment targets
keep their original storage.

These rules require expression-aware lowering. Moving checks into one prefix
would change execution. A function wrapper would change returns and defers.
Each source function therefore keeps its own return target.

## Keep types and names

Temporary variables use checked source types. In a `uint64` context,
`var saved uint64 = 1 << n` preserves the required conversion; an inferred
`int` temporary would change the program. Fresh names prevent collisions with
source bindings. Added import aliases include the package name, such as
`fmt_tgo3`; source import names stay unchanged. Required types must be nameable
in the generated scope.

The error check uses the call's result before conversion to the caller's return
type. A nil pointer can become a non-nil interface during that conversion.
Checking afterward would turn success into failure. `!` returns the original
error; `!!` adds call-name context with `%w`.

The private source checker handles propagation and `if` values together.
It checks branch values against the surrounding type and keeps branch
statements in the source function. The frontend's call markers identify added
syntax; they do not create source functions. This keeps returns and defers in
their original function.

In a generic call, another argument can supply the if value's type.
For `pair(if ok { nil } else { nil }, pointer)`, a pointer of type `*int`
determines `T` as `*int`. Infer from every branch that produces a value and
from the other arguments together. Then check each branch against the resolved
parameter type. This supplies nil's type and checks branch constants for
overflow. A branch that returns or jumps supplies no inference operand.

`make prepare` builds the checker and its compiled-package reader from one
pinned Go source version and checked-in patches. Generated upstream source
stays out of Git. See [the patch instructions](../../../internal/checker/upstream.md).
Their type objects stay separate from stock `go/types` objects. Public `Load`
keeps its stock package and checks the same syntax separately for lowering.
After lowering, stock Go checking checks the emitted module.

The function keeps its source result names. Failure returns use saved zero
values for results before the error. Deferred calls still run.

## Keep loop targets

An initializer can contain an if value that breaks or continues its loop.
Run that initializer on the loop's first visit. Later visits run the post
statement before checking the condition. The generated loop therefore remains
the target of both jumps.

Go creates a separate variable for each iteration of a loop declared with
`:=`. Keep those declarations in the generated Go loop header. Store the
initializer result on the first visit. Rename these bindings by checked object
identity: in `for i := if ok { break } else { i }; ...`, the final `i` in the
initializer still refers to the outer variable. Closures in the body refer to
the per-iteration variable. The rename copies emitted syntax and leaves the
checked source unchanged.

## Source controls output

Generated files can be stale, so they do not participate in source checking.
Compilation works in a separate directory and returns file contents. The
caller writes them.

`name.tgo` produces `name_tgo.go`. Test and platform suffixes stay last:
`name_linux_test.tgo` produces `name_tgo_linux_test.go`. Physical paths decide
file selection; line directives decide diagnostic locations.

See the [correctness report](correctness.md) for behavior checks and the
[user guide](../../guide/transpiler.md) for commands and API use.

## Prepare conditional syntax

Go's parser owns headers and statement grammar. The frontend first finds clear
expression operands, such as `return if` and `use(if ...)`. It checks each
conditional's bounded syntax fragment, then wraps independent operands together
in one buffer edit. The complete file is parsed after preparation. Token nesting
keeps inner operands ahead of their containing expressions.

A colon can introduce a label, map value, case body, or slice bound. For these
ambiguous positions, the frontend uses the parsed AST to identify an expression
operand. Nested expressions can need several preparation rounds. This preserves
Go's grammar and source positions; deeply nested syntax still costs more than
independent expressions.
