# Eager slice and map comprehensions

Status: accepted.

## Decision

Combine generators, filters, and outputs in direct loops with one result
collection. A list result is a Go slice. Use typed literals with the
output first:

```text
names := []string{u.Name for _, u := range users if u.Active}
byID := map[ID]User{u.ID: u for _, u := range users if u.Active}
pairs := [][2]int{[2]int{x, y} for _, x := range xs for _, y := range ys}
```

State the slice or map type. Write a slice element or map `key: value`, then
one or more `for` clauses and optional `if` filters. Use Go range bindings.
Explicit types give `nil` and untyped constants a target type.

One output, or one key and value, precedes at least one `for` clause. Further
`for` and `if` clauses nest in source order. Generators use `for range source`
or declare names with `:=`, under Go range type and variable-count rules.
Filters are Boolean expressions. Newlines can separate clauses; a final comma
is optional. Keep ordinary Go literals unchanged. Do not mix literal elements
with comprehension clauses.

## Alternatives and reason

Each candidate can become direct loops. Punctuation does not determine speed.

| Candidate | Example | Judgment |
| --- | --- | --- |
| Inferred, output first | `[u.Name for _, u := range users if u.Active]` | Shortest. Needs collection inference rules and new literal delimiters. |
| Typed, output first | `[]string{u.Name for _, u := range users if u.Active}` | Chosen. Keeps Go literal types and range bindings; makes the output explicit. |
| Forward with `yield` | `[]string{for _, u := range users if u.Active yield u.Name}` | Reads in execution order. Adds a `yield` role. Useful for long clause chains. |
| Nested blocks | `[]string{for _, u := range users { if u.Active { u.Name } }}` | Familiar Go loops. More nesting; extends final-expression rules to loop and filter blocks. |
| Library pipeline | `Collect(Map(Filter(users, active), name))` | No new syntax. Function arguments and boundaries add text; fusion depends on library rules. |

Maps use `key: value`. Typed forms retain `map[K]V{...}`; the inferred form
would be `{u.ID: u for _, u := range users}`. Inference is feasible, but needs
rules for nil, constants, context, and named collections. For example,
`[1 for range n]` can default to `[]int`; `[]uint64` needs context or a type.

Choose explicit collection types because Go users already state them in literals.
Keep `:= range` because
`for v in xs` needs a rule to select index, key, or value. Output-first syntax
makes the result visible without nested braces. Its cost is that execution
starts with clauses written after the output. Long chains can use newlines.
Inferred literals need more type rules. `yield` adds a keyword role. Nested
blocks add braces for each loop and filter. A pipeline needs function
arguments and separate rules to combine its operations into one loop.

| Research | Approach | Lesson for TGo |
| --- | --- | --- |
| [Python](https://docs.python.org/3/reference/expressions.html#comprehensions) | Output first, ordered generators and filters, separate binding scope. | State clause order and local scope. |
| [XGo lists](https://tutorial.xgo.dev/list-comprehension) and [maps](https://github.com/goplus/xgo/blob/main/doc/map.md#map-comprehensions) | Short inferred comprehensions in a Go-derived language. | Inference is an alternative, not a fusion requirement. |
| [Julia](https://docs.julialang.org/en/v1/manual/arrays/#Comprehensions) | Optional explicit element type and dependent generators. | Explicit types can coexist with concise syntax. |
| [Scala](https://docs.scala-lang.org/tour/for-comprehensions.html) | Forward generators, guards, `yield`; uses `map`, `flatMap`, `withFilter`. | Forward syntax is useful; library translation alone does not guarantee fusion. |
| [Rust](https://doc.rust-lang.org/std/iter/index.html#laziness) | Lazy adapters consumed by operations such as `collect`. | Combine steps before collection; TGo's result remains eager. |
| [GHC](https://ghc.gitlab.haskell.org/ghc/doc/users_guide/exts/rewrite_rules.html#list-fusion) | Producer and consumer rules remove intermediate lists. | Execution rules must permit collection removal. |

These comparisons do not establish a TGo speed result.

## Behavior

Execute where the expression occurs. Follow [Go range rules](https://go.dev/ref/spec#For_statements),
including the constant-length evaluation exception. Inner sources run only
when reached. A false filter skips later clauses and output for that visit.
Evaluate accepted outputs once. For maps, evaluate the key, then the value,
then insert. Later equal keys replace earlier entries; map iteration order
stays unspecified.

Names stay local. Later clauses and outputs can use earlier names. A generator's
own source uses the preceding scope. Preserve separate `:=` iteration variables.
Accept slice and map types, including named types, and use Go assignment rules.
Results are fresh and non-nil, even when empty; elements can refer to source
objects. Slice capacity is unspecified. Results are nonconstant values, with
Go composite literal address-taking rules.

Sources, filters, keys, and values can contain existing TGo expressions.
Propagation, returns, defers, and jumps keep their enclosing function and
targets. In filters and outputs, unlabeled `break` or `continue` targets the
innermost active generator. A source can target only already active enclosing
constructs. No helper encloses a function's comprehension. Package initializers
follow existing rules. Array `len`, `cap`, and `range` evaluate contained
comprehensions; unsafe size and offset operands remain unevaluated.

## Implementation and optimization

Extend the [frontend](../../internal/frontend/readme.md),
[checker](../../internal/checker/readme.md), and
[lowerer](../../internal/lowering/readme.md). Keep the
[driver stages](../../internal/driver/readme.md) and final stock Go check.

Record types, ordered clauses, outputs, and positions. Use an expression marker
for Go parsing; parse clause expressions with the frontend. Recognize boundaries
outside nested expressions and permit inserted semicolons there. Check generators
and filters in execution order, opening each range scope before later clauses.
Check outputs against element, key, and value types. Record bindings and jump
targets; mark the operation as evaluated for array `len`, `cap`, and `range`.
Placeholders must not create source function boundaries.

For `[]string{u.Name for _, u := range loadUsers()! if u.Active}`, the basic
output inside a function returning `([]string, error)` is:

```go
users, err := loadUsers()
if err != nil {
    return nil, err
}
result := make([]string, 0)
for _, u := range users {
    if u.Active {
        result = append(result, u.Name)
    }
}
names := result
```

Use fresh names, nested ranges, and filters enclosing later clauses. For maps,
evaluate typed key and value locals before insertion. Retain range headers for
array copies, iterator functions, map mutation, and iteration bindings. Use
existing expression lowering at each evaluation site. Save earlier operands;
preserve branches, loop visits, assignment destinations, returns, and defers.
Generate address-taking storage and retain outer jump targets. Extend package
initializer lowering under its existing wrapper rules.

Fusion within a comprehension is required: no intermediate collections or
per-element callback wrappers. Nested generators still require nested loops.
[Python's inlined comprehensions](https://peps.python.org/pep-0709/) support
direct execution without a comprehension call frame.

For one slice or array range with one output per visit, allocate the result
length once and write by index. Prove filters and control flow cannot change
the count; otherwise start empty and append. Supply a map size hint only when
distinct entry count is proved. Otherwise use normal growth. Do not reserve
the entire input for sparse filters or repeated keys, run sources twice, count
in a separate pass, force elided operands, or multiply unchecked nested bounds.
One destination does not guarantee one heap allocation.

Keep distinct eager comprehensions in source order, including comprehension
range sources. Producing `f(a), f(b)` before `g(a), g(b)` differs from alternating
`f(a), g(a), f(b), g(b)`. Errors, panics, mutation, and aliases expose that
difference. Map replacement also occurs before consumers see final entries.

Follow [modernization rules](../problem/README.md#adoption-and-automatic-modernization).
A nil accumulator differs from the non-nil empty result. Observed capacity or
a visible partial destination can prevent an automatic loop replacement.

## Validation

`TestCollectionComprehensions` compiles `.tgo` fixtures, compares generated Go
with committed snapshots, and runs both the output and independent Go loops
against `stdout.txt`.

The fixtures in `tests/testdata/comprehensions/` check:

- `Ranges`: every Go range kind, dependent generators, filters, Unicode,
  nil inputs, closure capture, and iterator cleanup on break.
- `Types`: named and generic collections, contextual shifts, nil and interface
  outputs, address-taking, shadowed bindings, and non-nil empty results.
- `Effects` and `Operations`: source call counts, separate eager collections,
  key-before-value execution, duplicate keys, array copies, map mutation,
  stores, go and defer arguments, and select operands. An unhashable interface
  key must evaluate its value before insertion panics.
- `Control`: original and wrapped errors, source returns and defers, break,
  continue, goto, and jumps from a generator source to an active outer loop.
- `Builtins`: array len, cap, and range execute contained comprehensions;
  unsafe size, alignment, and offset operands remain unevaluated.
- `HiddenBuiltins`: ordinary Go declarations named append, make, and len stay
  unchanged while generated operations keep their meaning.

`TestComprehensionErrors` checks invalid types, bindings, filters, output counts,
overflow, scope, and unsafe propagation. Diagnostics identify `.tgo` positions.
Frontend tests check comments, multiline syntax, and ordinary loops.

Snapshots show one destination, the exact-length and append paths, and map
size hints. Effect traces check execution order independently of these snapshots.

The `Benchmarks` fixture compares generated code with handwritten Go for dense
and sparse filters, nested generators, repeated keys, and iterators. It also
measures a materialized filter-then-transform loop. Run:

```sh
go -C tests/testdata/comprehensions/Benchmarks test -run '^$' -bench . -benchmem
go test ./internal/driver -run '^$' -bench BenchmarkCompileComprehensions
```

The compiler benchmark measures independent and nested comprehensions at 1,
16, and 64 expressions. Compare allocations and bytes as well as elapsed time.
