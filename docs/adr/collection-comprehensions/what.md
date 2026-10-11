# WHAT

Status: proposed. This feature is not implemented.

Add eager slice and map comprehensions. A list result is a Go slice. Each
comprehension combines iteration, filters, and output expressions in direct
loops that write into one result collection.

## Syntax

Use typed literals with the output first:

```text
names := []string{u.Name for _, u := range users if u.Active}
byID := map[ID]User{u.ID: u for _, u := range users if u.Active}
pairs := [][2]int{[2]int{x, y} for _, x := range xs for _, y := range ys}
```

The type and output are explicit. Use Go range bindings for each generator.
[SYNTAX](syntax.md) records the decision and compares alternatives.

One output, or one key and value, precedes at least one `for` clause. Further
`for` and `if` clauses nest in source order. Generators use `for range source`
or declare names with `:=`, under Go range type and variable-count rules.
Filters are Boolean expressions. Newlines can separate clauses; a final comma
is optional. Keep ordinary Go literals unchanged. Do not mix literal elements
with comprehension clauses.

## Behavior

Execute where the expression occurs. Follow [Go range rules](https://go.dev/ref/spec#For_statements),
including the constant-length evaluation exception. Inner sources run only
when reached. A false filter skips later clauses and the output for that
visit. Evaluate accepted outputs once. For maps, evaluate the key, then the
value, then insert. Later equal keys replace earlier entries. Map iteration
order stays unspecified.

Names stay inside the comprehension. Later clauses and outputs can use earlier
names. A generator's own source uses the preceding scope. Preserve Go's
separate variables for each `:=` iteration.

Accept slice and map types, including named types. Use Go assignment rules
for outputs. Results are fresh and non-nil, even when empty; elements can
still refer to source objects. Slice capacity is unspecified. Results are
nonconstant values. Address-taking follows Go composite literal rules.

Sources, filters, keys, and values can contain existing TGo expressions.
`!`, `!!`, returns, defers, and jumps retain their enclosing function and
checked control-flow targets. In filters and outputs, an unlabeled `break`
or `continue` targets the innermost active generator. A generator's source can
target only already active enclosing constructs. No helper function encloses
a function's comprehension. Package initializers follow existing rules.
Array `len`, `cap`, and `range` evaluate a contained comprehension; unsafe
size and offset builtins keep their unevaluated operands.

[HOW](how.md) defines lowering and optimization. [PROOF](proof.md) defines
validation.
