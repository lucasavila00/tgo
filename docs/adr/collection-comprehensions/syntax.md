# SYNTAX

## Decision

Use typed collection literals with the output before the clauses:

```text
names := []string{u.Name for _, u := range users if u.Active}
byID := map[ID]User{u.ID: u for _, u := range users if u.Active}
```

State the slice or map type. Write a slice element or map `key: value`, then
one or more `for` clauses and optional `if` filters. Use Go range bindings.

This form keeps Go's collection types and makes the result easy to find.
Explicit types also give `nil` and untyped constants a target type. Clauses
execute first, although the output appears first. Use multiple lines for long
clause chains.

## Alternatives

The examples select active users and produce their names.

| Candidate | Example | Judgment |
| --- | --- | --- |
| Inferred, output first | `[u.Name for _, u := range users if u.Active]` | Shortest useful form. Needs collection type inference and new literal delimiters. |
| Typed, output first | `[]string{u.Name for _, u := range users if u.Active}` | Chosen. Keeps Go literal types and range bindings; makes the output explicit. |
| Forward clauses with `yield` | `[]string{for _, u := range users if u.Active yield u.Name}` | Reads in execution order. Adds a `yield` role and keeps the type prefix. Better for long clause chains. |
| Nested blocks | `[]string{for _, u := range users { if u.Active { u.Name } }}` | Closest to Go loops. More nesting for a small operation; extends the final-expression rule to loop and filter blocks. |
| Library pipeline | `Collect(Map(Filter(users, active), name))` | Can work without new syntax. Function arguments add text and function boundaries; fusion depends on library execution rules. |

Map variants use `key: value` as the output. Typed variants retain
`map[K]V{...}`. The inferred form would be `{u.ID: u for _, u := range users}`. Most
outputs have an obvious type. However, `nil`, untyped constants, contextual
types, and named collections still need rules. For example, standalone
`[1 for range n]` can default to `[]int`; producing `[]uint64` needs context
or an explicit type. This is feasible, but it is a separate design choice.

Choose explicit types because Go users already state them in collection
literals. Keep `:= range` because `for v in xs` must decide whether `v` is an
index, key, or value. Inferred literals need more type rules. `yield` adds a
keyword role. Nested blocks add braces for each loop and filter. A pipeline needs function
arguments and separate rules to combine its operations into one loop.

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
