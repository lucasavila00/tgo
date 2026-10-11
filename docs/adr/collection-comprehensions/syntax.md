# SYNTAX

Choose syntax for users first. Every candidate below can become direct loops;
no candidate is faster because of its punctuation.

## Alternatives

The examples select active users and produce their names.

| Candidate | Example | Judgment |
| --- | --- | --- |
| Inferred, output first | `[u.Name for _, u := range users if u.Active]` | Shortest useful form. Needs collection type inference and new literal delimiters. |
| Typed, output first | `[]string{u.Name for _, u := range users if u.Active}` | Clear output and type; few braces. Keeps Go literal types and range bindings. Recommend. |
| Forward clauses with `yield` | `[]string{for _, u := range users if u.Active yield u.Name}` | Reads in execution order. Adds a `yield` role and keeps the type prefix. Better for long clause chains. |
| Nested blocks | `[]string{for _, u := range users { if u.Active { u.Name } }}` | Closest to Go loops. More nesting for a small operation; extends the final-expression rule to loop and filter blocks. |
| Library pipeline | `Collect(Map(Filter(users, active), name))` | Can work without new syntax. Function arguments add text and function boundaries; fusion depends on library execution rules. |

Map variants use `key: value` as the output. Typed variants retain
`map[K]V{...}`. The inferred form would be `{u.ID: u for _, u := range users}`. Most
outputs have an obvious type. However, `nil`, untyped constants, contextual
types, and named collections still need rules. For example, standalone
`[1 for range n]` can default to `[]int`; producing `[]uint64` needs context
or an explicit type. This is feasible, but it is a separate design choice.

Recommend explicit types because Go users already state them in collection
literals. Keep `:= range` because `for v in xs` must decide whether `v` is an
index, key, or value. The output-first form reduces nesting and makes the
result shape visible. Its cost is that execution starts with the clauses,
although the output appears first. Long chains can use multiple lines.

## What other languages teach us

| Source | Approach | Use for TGo |
| --- | --- | --- |
| [Python](https://docs.python.org/3/reference/expressions.html#comprehensions) | Output first; ordered generators and filters; separate binding scope. | Keep clause order and local bindings explicit. |
| [XGo lists](https://tutorial.xgo.dev/list-comprehension) and [maps](https://github.com/goplus/xgo/blob/main/doc/map.md#map-comprehensions) | Short, inferred comprehensions in a Go-derived language. | Inference is a real alternative, not a requirement for fusion. |
| [Julia](https://docs.julialang.org/en/v1/manual/arrays/#Comprehensions) | Output first, optional explicit element type, dependent generators. | Explicit types and concise comprehensions can coexist. |
| [Scala](https://docs.scala-lang.org/tour/for-comprehensions.html) | Forward generators, guards, and `yield`; uses `map`, `flatMap`, and `withFilter`. | Execution-order syntax is useful; library translation alone gives no TGo fusion guarantee. |
| [Rust](https://doc.rust-lang.org/std/iter/index.html#laziness) | Lazy iterator adapters, consumed by operations such as `collect`. | Combine operations before collection; keep TGo's proposed result eager. |
| [GHC](https://ghc.gitlab.haskell.org/ghc/doc/users_guide/exts/rewrite_rules.html#list-fusion) | Producer and consumer rules remove intermediate lists. | Eliminating collections requires execution rules that permit it. |

These are design comparisons. They do not establish a TGo speed result.
