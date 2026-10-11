# PROOF

Create positive and negative fixtures in
`compilerv2/testdata/initialization/`. Record each rejected source position.
Compile and run accepted output against handwritten Go references.

| Operations | Cases |
| --- | --- |
| Declarations | Local/package/grouped/blank; tuples; explicit zero/nil |
| Function results | Reject names in declarations, literals, nested types |
| Literals | Missing fields/indices; nesting; embedding; aliases; empty values |
| Allocation | Reject `new(T)`; accept `new(value)`; slice length/capacity |
| Mutation | Slice/map `clear`; ordered stores; pointer aliases; escapes |
| Collections | `append`, `copy`, comprehensions; initialized/uninitialized growth |
| Reslicing | Shorten then extend; alias writes; offsets; guarded bounds |
| Presence | Map miss, closed receive, assertions; success/failure paths |
| Errors | Pending tuples; forwarding; discarded results; failure commas |
| Control flow | Joins, loops, post, select, fallthrough, labels, goto |
| Generics | Mixed constraints; nested containers; inferred/explicit arguments |
| Generic effects | Wrappers, methods, returned closures, function values |
| Foreign calls | Inferred/supplied/missing contracts; alias and mutation effects |
| Dispatch | Interface methods; multiple targets; unavailable target contract |

Require primitive types to fail the same default-construction cases as models.
A generic `new(T)` must fail even when instantiated with `int`. Explicit
initialized values must remain accepted. A failed map/error value must not become usable
through package storage, pointers, closures, or copied assignments. Accept
copied status tests, delayed safe closure execution, forwarded pairs, and use
after a stopping failure branch.

Prove initialized reslices through prior population, slice aliases, and guarded
bounds. Reject capacity-only growth and facts invalidated by foreign writes.
Test generic effects through multiple package boundaries and stored closures;
unknown effects must not become safe by omission.

Compare evaluation order/count, skipped work, panic timing, return-expression
and defer timing, exact `!!` identity, and single `!` wrapping. Returned pointers
can expose deferred writes; returned scalars retain their evaluated values.
Generated failure storage must never acquire a source initialization fact.

Imported contract tests must distinguish compiler enforcement from foreign
promises: missing contracts fail compilation; a supplied false promise is
outside the guarantee and adds no runtime guard.

```sh
go -C compilerv2 test ./... -run '^TestInitialization' -count=1
```

Run full validation in hosted CI.
