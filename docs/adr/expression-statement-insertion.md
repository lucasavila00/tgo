# WHAT: insert a return at expression evaluation

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Behavior

- Select a Go expression and insert `return err` before or after it evaluates.
- Before: return before evaluating that expression or its operands.
- After: evaluate it once, then return before later work.
- If execution skips the expression, skip the return.
- Return from the source function, with the same error and normal defers.
- Preserve Go evaluation rules, types, bindings, scopes, and control flow.
- Cover declarations, assignments, calls, short-circuit operands, branches,
  loop headers and post statements, ranges, switches, and selects.
- Distinguish immediate operand evaluation from scheduled `go` / `defer` calls.

## Evidence

- Compare executable traces with handwritten Go, including skipped work,
  repeated execution, panic timing, labels, and nested functions.
- Keep the experiment separate from production and from `!` / `!!` parsing.
- Review [HOW](expression-statement-insertion-how.md) separately. A proposed
  implementation does not establish that every requested boundary is feasible.
