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
| `internal/checker` | Check source with Go types and TGo rules | Nested operators need success types before lowering. |
| `internal/frontend` | Record added syntax | Go syntax keeps Go meaning; TGo operations retain their source identity. |
| `internal/lowering` | Emit equivalent Go statements | This requires checked types and the expression's execution context. |

The driver selects packages and asks the private checker to check their source.
The checker reads frontend markers. Lowering uses the checked types and markers.
The public API retains stock Go package objects.

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

The private checker gives `call()!` the call's success result types. For
`outer(inner()!)!`, it checks the inner call before checking the outer call.
Each source package is checked once; operators need no file rewrite or package
reload. The driver then lowers the source and checks the emitted Go.
Source checks also verify error return types and forbidden unsafe operands.

`make prepare` applies tracked patches to pinned Go checker and import-reader
source. Generated upstream files stay outside Git. See the
[upstream source instructions](../../../internal/checker/upstream.md).

The function keeps its source result names. Failure returns use saved zero
values for results before the error. Deferred calls still run.

## Source controls output

Generated files can be stale, so they do not participate in source checking.
Compilation works in a separate directory and returns file contents. The
caller writes them.

`name.tgo` produces `name_tgo.go`. Test and platform suffixes stay last:
`name_linux_test.tgo` produces `name_tgo_linux_test.go`. Physical paths decide
file selection; line directives decide diagnostic locations.

See the [correctness report](correctness.md) for behavior checks and the
[user guide](../../guide/transpiler.md) for commands and API use.
