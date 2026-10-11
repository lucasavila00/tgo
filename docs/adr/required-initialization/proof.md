# PROOF

Create fixtures in `compilerv2/testdata/initialization/`. Negative fixtures
record the rejected source position. Compile and run positive output.

| Cases | Check |
| --- | --- |
| Local, package, grouped, blank vars | Reject missing initializers |
| Later assignment on every path | Still reject missing initializer |
| Tuple initialization, shadowing | Accept; preserve order and binding |
| Named and `_` results | Reject declarations, literals, function types |
| Nested function types, aliases, interface methods | Reject named results |
| Imported Go named results | Accept calls and function assignments |
| Complete, missing, embedded, nested fields | Check literal completeness |
| Keyed arrays/slices, inferred length, gaps | Check index coverage |
| Empty structs/arrays/slices/maps | Accept empty values |
| Aliases, generic literal types | Apply the same literal checks |
| Explicit return, void bare return, value bare return | Accept, accept, reject |
| Initialized deferred captures | Accept local writes and reference changes |
| Allocation, map misses, closed channels | Preserve Go behavior |
| Generic failure zeros, `!`, `!!` | Compile; preserve wrapping and identity |

Compare side-effect traces with handwritten Go references. Check initializer
and literal evaluation order. Check return-expression evaluation before defers:
a deferred local write leaves a returned scalar unchanged; a pointee write is
visible through a returned pointer. Generated locals must not trigger source
checks. Existing non-nil and checked-value rules must still apply.

```sh
go -C compilerv2 test ./... -run '^TestInitialization' -count=1
```

Run full validation in hosted CI.
