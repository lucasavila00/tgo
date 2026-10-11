# PROOF

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Committed corpus

Create these files under `spikes/expressionlowering/testdata/proof/`.
They are planned fixtures, not existing tests.

| File | Required cases |
| --- | --- |
| `calls.go` | Nested calls, tuples, discarded results |
| `booleans.go` | Both short-circuit paths, named Boolean types |
| `declarations.go` | `:=`, grouped declarations, shadowing |
| `assignments.go` | Arrays, pointers, maps, ordered stores |
| `loops.go` | Headers, post, continue, captured iteration variables |
| `switches.go` | Conditional cases, fallthrough, type bindings |
| `ranges.go` | Skipped array evaluation, later target failure |
| `selects.go` | Send operands, receives, selected targets |
| `scheduling.go` | Go calls, defers, direct recover |
| `jumps.go` | Goto, labeled loops, switches, selects |
| `types.go` | Constants, generics, methods, pointer receivers |
| `nested.go` | Closures and separate return targets |
| `foreign/foreign.go` | Imported calls and private named types |

## Matrix

- Inventory every AST expression by file, byte offsets, and node kind.
- Commit `manifest.json`: target functions, inputs, sites, classifications.
  Target functions return `error`; ordinary helpers supply expression values.
- Generate separate before and after variants for every runtime site.
- Test conditional sites with reaching and skipping inputs.
- Report type syntax, statically unevaluated operands, incompatible signatures,
  and unrepresentable boundaries explicitly. Never silently skip a node.
- **Neutral:** insert a test-only marker statement. Remove marker events from
  observations; require original behavior. Also compare marker positions with
  independent expected positions.
- **Return:** insert `return err`. For repeated sites, use a test-only guard
  that triggers on the specified visit. Check first and later visits.

## Independent checks

- Commit handwritten `reference.go` and `expect.json`; never generate expected
  results with the lowering code.
- Check traces, mutations, results, error identity, panics, and defer order.
  Skipped sites must preserve baseline behavior. Callers can continue after
  an injected function return.
- Use bounded inputs and synchronized channels. Test each select branch with
  one ready case. Compare permitted outcomes where Go leaves order unspecified.
- Every variant must compile. Missing sites, wrong placement, unexpected
  diagnostics, or changed observations fail the suite.

## Planned tests

- `TestProofInventory`: every node accounted for.
- `TestProofNeutral`: behavior and marker placement.
- `TestProofReturn`: early exit, visit count, error identity, defers.
- `TestProofCompile`: every generated variant.

```sh
go -C spikes/expressionlowering test ./... -run '^TestProof' -count=1
```

Run the full matrix in hosted CI; retain per-site results as an artifact.
