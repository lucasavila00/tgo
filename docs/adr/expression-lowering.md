# Add statements at an expression's evaluation point

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Decision and reason

Error propagation needs statements inside larger expressions. Emit those
statements where the source expression executes. Preserve operand order,
skipped paths, assignment storage, and enclosing return targets.

For `use(first(), target()!, last())`, save `first` before calling `target`.
Failure must return before `last` or `use` executes. For `left() && right()!`,
the right call and its check belong inside the true path of `left`.
The same placement rule supports ordinary-Go `InsertAfter` and TGo lowering.

Read checked syntax and build changed nodes without modifying the input.
Format output with `go/format`. The [source-checking decision](source-checking.md)
supplies private TGo types; the driver writes output and stock Go checks it.
See [lowering](../../internal/lowering/readme.md) for current implementation
details and examples.

## Proof corpus

Under `tests/testdata/proof/`, the corpus covers:

| File | Behavior |
| --- | --- |
| `calls.go` | Nested calls, tuples, discarded results |
| `booleans.go` | Reached and skipped operands, named Boolean types |
| `declarations.go` | Declarations, grouping, shadowing |
| `assignments.go` | Arrays, pointers, maps, store order, panics |
| `loops.go` | Headers, post work, continue, captured variables |
| `switches.go` | Case order, fallthrough, type bindings |
| `ranges.go` | Skipped evaluation and later destination failure |
| `selects.go` | Sends, receives, destinations |
| `scheduling.go` | Go calls, defers, recover |
| `jumps.go` | Goto and labeled jumps |
| `types.go` | Constants, generics, methods, pointer receivers |
| `nested.go` | Closures and separate return targets |
| `foreign/foreign.go` | Imported calls and private types |

## Validation

`manifest.json` identifies sites by file and byte offsets. It also marks
unevaluated syntax and functions that cannot accept an inserted error return.
For each eligible site, check two rewrites:

1. Insert a neutral trace marker. Remove marker events and require the original
   behavior. Marker positions must match handwritten traces.
2. Insert `return err`. Check the exact early exit, including later loop visits.

Handwritten `reference.go` and `expect.json` specify effects, values, errors,
panics, and defers. They must not use compiler lowering code. Every accepted
variant must compile. Channel and select cases use synchronization and one ready
case, while permitting orders Go leaves unspecified.

`rejections.json` lists inaccessible saved types. Retain those sites in the
inventory and require their specified diagnostics. Other eligible sites must
compile and preserve behavior.

`TestProofInventory`, `TestProofNeutral`, `TestProofReturn`, and `TestProofCompile`
perform these checks. See [tests](../../tests/readme.md) for fixture and CI use.
