# PROOF

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Files to create

Under `compilerv2/testdata/proof/`:

| File | Cases |
| --- | --- |
| `calls.go` | Nested calls, tuples, discarded results |
| `booleans.go` | Reached and skipped operands, named Boolean types |
| `declarations.go` | `:=`, grouped declarations, shadowing |
| `assignments.go` | Arrays, pointers, maps, store order, panics |
| `loops.go` | Headers, post, continue, captured variables |
| `switches.go` | Case order, fallthrough, type bindings |
| `ranges.go` | Skipped evaluation, later target failure |
| `selects.go` | Send operands, receives, targets |
| `scheduling.go` | Go calls, defers, recover |
| `jumps.go` | Goto and labeled control flow |
| `types.go` | Constants, generics, methods, pointer receivers |
| `nested.go` | Closures, separate return targets |
| `foreign/foreign.go` | Imported calls, private types |

## Automatic checks

- Inventory every expression by file and byte offsets in `manifest.json`.
  Record non-runtime expressions and incompatible return signatures explicitly.
- For each expression evaluated in the current function, generate an insertion
  immediately after it. Test reached and skipped paths.
- **Neutral marker:** require unchanged behavior after removing marker events.
  Check marker positions against handwritten expected traces.
- **Return:** insert `return err`; check the expected early exit. A test-only
  guard selects the first or a later visit for expressions inside loops.
- Commit handwritten `reference.go` and `expect.json`. Do not generate either
  with the compiler under test.
- Compare effects, results, error identity, panics, and defers. Compile every
  generated variant.
- Use synchronized channels and one ready select case. Accept the orders that
  Go permits where evaluation order is unspecified.

## Planned command

```sh
go -C compilerv2 test ./... -run '^TestProof' -count=1
```

Tests: `TestProofInventory`, `TestProofNeutral`, `TestProofReturn`,
`TestProofCompile`. Run the full matrix in hosted CI; retain per-site results.
