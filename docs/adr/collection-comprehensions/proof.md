# PROOF

Implementation must pass source checks, generated-Go checks, and runtime
comparisons with independent handwritten Go loops. Run validation in hosted
CI. This ADR does not change compiler behavior.

Check:

- Slice and map syntax, named types, type parameters, comments, multiline
  clauses, nesting, and diagnostics at original source positions. Ordinary
  literals must retain their parser behavior.
- Every Go range kind, valid binding counts, nil inputs, the constant-length
  evaluation exception, Unicode strings, map mutation, and iterator cleanup.
- Dependent generators, filters before inner sources, and source call counts.
  Compare effect traces, not only final collections.
- Empty non-nil results, fresh result storage, address-taking, duplicate keys,
  key-before-value execution, and an interface key that causes a comparison panic.
- Local scope, shadowing in a generator's own source, closure capture of
  separate iteration variables, typed nil, constants, and overflow errors.
- `!`, `!!`, if values, returns, defers, labeled jumps, surrounding operands,
  assignments, short-circuit expressions, loops, package initialization, and
  evaluated versus unevaluated builtin operands.
- Separate comprehensions, including comprehension range sources, with visible
  effects and duplicate map keys. Keep production and consumption phases separate.

Snapshots must show one destination and no intermediate filter or transform
collections. Check both the append path and proven exact-length path.

Benchmark against handwritten Go and a materialized filter-then-transform
reference. Include dense and sparse filters, large inputs, nested generators,
maps with repeated keys, and unknown-length iterators. Report runtime,
allocations, and bytes allocated. Include compiler time for many independent
and deeply nested comprehensions. Do not claim a speed gain before measurement.

[WHAT](what.md) defines the feature. [HOW](how.md) defines the implementation.
