# PROOF

After approval, create positive and negative fixtures under
`compilerv2/testdata/initialization/`. Each negative fixture records the source
position and expected rule. Compile positive output with Go and run cases
where output text cannot prove behavior.

| File | Required cases |
| --- | --- |
| `declarations.tgo` | Local/package/grouped/blank vars; tuples; shadowing |
| `bindings.tgo` | Parameters, receivers, range, select, type switch |
| `literals.tgo` | Complete/missing fields; embedding; nested values; aliases |
| `collections.tgo` | Array/slice gaps, inferred length, empty values, maps |
| `results.tgo` | Named results, explicit/bare returns, partial field stores |
| `branches.tgo` | Both/one branch assigns; terminating branch; switches |
| `loops.tgo` | Zero iterations, continue/post, break, labels, goto |
| `closures.tgo` | Capture/address before assignment; deferred reads/writes |
| `generics.tgo` | Generic results, constrained literals, explicit allocation |
| `interop.tgo` | Imported private fields, factory calls, zero-valued Go calls |
| `errors.tgo` | Failure commas, propagation, checked failure values |

Verify that named results do not obtain assignment from a loop's later
iteration, an unreachable store, a called helper, or a closure body. Include
select receive assignment and switch fallthrough. Verify that explicit return
values become visible to a previously registered defer and that the defer can
change the returned values. Initialize captured results before registration.

For valid cases, compare side-effect traces with handwritten Go references:
initializer order, tuple assignment, keyed literal evaluation, panic timing,
allocation, failure return, and deferred calls. Verify exact `!!` error
identity and single `!` wrapping. Generated locals from expression lowering
must not produce source-initialization errors.

Test explicit zeros, nullable nil, map misses, closed-channel receives,
`new(T)`, `new(expression)`, and slice length/capacity separately. Existing
non-nil and checked-value checks must still reject invalid construction.
Generic failure zeros must compile without requiring user-written helpers.

Planned focused command:

```sh
go -C compilerv2 test ./... -run '^TestInitialization' -count=1
```

Run full validation in hosted CI. This checkout has no compiler or CI; this
ADR adds no test runner. Implementation must add these tests, update README
examples and language documentation, and record compatibility changes before
completion. Approval of this proposal is not proof of implementation.

[WHAT](what.md) states the rules. [HOW](how.md) states the algorithm.
