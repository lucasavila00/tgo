# Transpiler architecture

The transpiler reads TGo and produces Go. It uses Go's parser and type checker.
The [API guide](../transpiler.md) explains how to use it.

## Packages

| Package | Work |
| --- | --- |
| `cmd/tgo` | Read command arguments and write output files. |
| `pkg/transpiler` | Provide the public compiler and statement-insertion API. |
| `internal/driver` | Load modules, select files, check types, and run compilation. |
| `internal/frontend` | Read `!` and `!!` and prepare source for Go parsing. |
| `internal/lowering` | Replace expressions with Go statements that keep their behavior. |

The public API calls the driver and lowering packages. The driver calls the
frontend and lowering packages. Neither frontend nor lowering imports the other.

## Compilation

1. Read the module and select files with Go's build rules.
2. Replace each postfix operator with a marker that Go can parse.
3. Use temporary helper calls to check the success values without the final
   error result. These helpers are for type checking, not error handling.
4. Replace each propagation operation with a call, an error check, and a return.
5. Format and write the Go files. Remove helpers that are no longer needed.

`name.tgo` produces `name_tgo.go`. Test and platform suffixes stay at the end:
`name_test.tgo` produces `name_tgo_test.go`. Existing generated files do not
participate in source type checking. See the API guide for module output rules. Line directives change diagnostic locations. File
selection and output use physical source paths.

## Error propagation

`call()!!` returns the same error from the function that contains it.
`call()!` wraps that error once with the source call name. On failure, later
statements do not run. Deferred calls still run.

The failure check uses the call's final result before return conversion. A nil
pointer, slice, map, channel, or function is success. A nil error interface is
success, but an interface that contains a nil pointer is failure. A result
whose type cannot be nil is failure, including a zero-valued concrete error.
A generic result follows these rules for its instantiated type. The compiler
uses reflection when Go cannot compare that result directly with nil. It checks
the result's declared type without extracting an interface's dynamic value.

Lowering visits named functions and function expressions, including functions
inside package variable initializers and collections. Each function has its
own return target. The surrounding package initializer keeps its order.

The compiler checks that every accepted propagation operation was processed.
If one was skipped, compilation fails instead of emitting a helper that
silently discards its error.

Both operators are forbidden anywhere inside operands of `unsafe.Sizeof`,
`unsafe.Alignof`, and `unsafe.Offsetof`.

## Statement insertion

`Load` returns typed source and a list of expression positions.
`InsertAfter` adds statements after a selected expression runs, before its
parent uses the result. Its callback receives the saved result values.

For example, insert `return err` after `right()` in `use(left(), right())`:

```go
leftValue := left()
rightValue := right()
return err
use(leftValue, rightValue)
```

Earlier work stays before the insertion. Skipped expressions stay skipped.
The last call remains in the syntax tree, but cannot run after the return.

A label on a declaration stays on the first added statement. The remaining
statements stay in the original scope, so later code can still use the source
variable. Other statements keep blocks when they need a separate scope.

The compiler copies changed syntax nodes. It keeps the original source tree
and type information, so callers can request another transformation.

## Types and storage

Added variables must keep source types, bindings, and evaluation order.
Private imported types and generic types need care because their names may
not be available where statements are added.

Array writes must use the original array. Slice and map operations must use
the correct saved values. Map key checks must not run before their source
position. Short-circuit expressions must not execute skipped operands.

## Tests

Integration fixtures are in `tests/`. They compare TGo output with committed
Go snapshots and check behavior. Independent Go references check call order,
results, errors, panics, and deferred calls. Package unit tests check individual
operations.

The expression proof tests insert statements at each recorded expression and
compare the results with handwritten expectations. See the
[test guide](../../tests/README.md) and
[proof plan](../adr/expression-lowering/proof.md).

## Typed temporaries

Every generated temporary that saves an expression uses an explicit `var`
type from the checked source. Tuple results use separate typed declarations
and one assignment, so the call runs once. Contextual shifts keep the type
supplied by their call, assignment, return, or field.

The compiler rejects a required temporary when Go cannot name its type.
This includes private imported types and type names hidden by source bindings.
The error identifies the source position and type. Expose an accessible type
at that boundary. There is no inferred temporary fallback.

The Boolean lowerer uses a typed local and assigns each operand only when
the source evaluates it. Comparisons keep their untyped Boolean behavior
when the surrounding operation supplies a named Boolean type.
