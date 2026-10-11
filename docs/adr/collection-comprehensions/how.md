# HOW

Extend the existing [frontend, private source checker, and lowerer](../../arch/transpiler/readme.md).
Keep the driver stages and stock Go validation.

## Parse and check source

Record the type, ordered clauses, outputs, and positions in the frontend.
Use an expression marker for Go's parser. Parse clause expressions with the
existing frontend. Recognize boundaries outside nested expressions and permit
inserted semicolons there. Keep ordinary literal parsing unchanged.

Check the type, then generators and filters in execution order. Open each
range scope before checking later clauses. Check outputs against the declared
element, key, and value types. Record bindings and jump targets. Mark the
operation as evaluated for array `len`, `cap`, and `range`. Placeholders must
not create a source function boundary.

## Emit direct loops

For example:

```text
names := []string{u.Name for _, u := range loadUsers()! if u.Active}
```

Inside a function returning `([]string, error)`, the basic output is:

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

Generate fresh names, nested ranges, and filters that enclose later clauses.
For maps, evaluate typed key and value locals in that order, then insert.
Keep range headers so array copies, iterator functions, map mutation, and
per-iteration bindings retain Go behavior.

Use existing expression lowering for sources, filters, and outputs. Save
earlier surrounding operands. Preserve branches, loop visits, assignment
destinations, returns, and defers. Generate storage for address-taking.
Generated loops must not capture jumps to outer targets. Extend package
initializer lowering under its existing wrapper rules.

## Fuse and allocate

Fuse clauses into one destination. Generate no intermediate collections or
per-element callback wrappers. Nested generators still require nested loops.
[Python's inlined comprehensions](https://peps.python.org/pep-0709/) support
this choice of direct execution without a comprehension call frame.

For one slice or array range with one output per visit, allocate the result
length once and write by index. Prove that filters and output control flow
cannot change the count. Otherwise start empty and append. Supply a map size
hint when its distinct entry count is proved; otherwise use normal Go growth.
Do not reserve the whole input length for sparse filters or repeated keys.
Never run a source twice, count it in a separate pass, force an elided range
operand, or multiply unchecked nested bounds. This guarantees one destination,
not one heap allocation in every case.

Keep distinct eager comprehensions in source order, including one used as
another's range source. Producing `f(a), f(b)` before consuming with
`g(a), g(b)` differs from alternating `f(a), g(a), f(b), g(b)`. Errors, panics,
mutation, and aliasing can expose that difference. A map also applies
duplicate-key replacement before a later consumer sees its entries.

Follow the [modernization rules](../../problem/README.md#adoption-and-automatic-modernization)
when replacing handwritten loops. A nil slice accumulator differs from this
feature's non-nil empty result. Observed capacity and a visible partially
built destination can also prevent an automatic replacement.

[WHAT](what.md) specifies behavior. [PROOF](proof.md) checks the result.
