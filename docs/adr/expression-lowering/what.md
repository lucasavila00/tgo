# WHAT

Issue: [#248](https://github.com/lucasavila00/tgo/issues/248)

## Transformation

- Lower expressions into ordinary Go statements so `return err` can execute
  immediately before or after a selected expression.
- Before: return before its operands evaluate. After: evaluate it once, then
  return. A skipped expression must not cause a return.
- Return from the source function with the same error and normal deferred calls.

## Algorithm

- For each expression, produce statements and replacement expressions.
- Save earlier pending evaluations in temporary variables before emitting a
  later operand's statements. Rebuild the parent from those saved values.
- Put generated statements where the expression executes, including conditional
  branches. Do not collect every statement into one unconditional prefix.
- Keep constants, contextual conversions, result counts, and assignment storage
  intact. Keep source bindings and scope; generate fresh names.

Example: insert after `right()` in `use(left() && right())`:

```go
r := left()
if r {
    r = right()
    return err
}
use(r)
```

## Construct rules

- **Declarations:** preserve `:=` scope and grouped declaration order. Keep
  source labels before generated declarations, so jumps remain valid.
- **Assignment:** prepare target operands, evaluate RHS values, then store
  left to right. Do not copy array storage or advance bounds panics.
- **Loops:** preserve initializer scope, per-iteration bindings, condition
  frequency, and post execution after `continue` but not after `break`.
- **Switches:** retain comparison types, conditional case evaluation,
  fallthrough, and break targets. Preserve type-switch bindings.
- **Ranges:** retain native skipped-expression rules and iteration bindings;
  evaluate assignment targets on each iteration.
- **Selects:** prepare operands before selection; evaluate receive targets
  afterward. An after-receive return precedes target evaluation.
- **Go/defer:** evaluate operands now, schedule the call normally. After-root
  insertion follows scheduling, not eventual call execution.
- **Nested functions:** preserve separate return targets.

## Boundary

- Native Go cannot insert a statement between select selection and
  communication. Report that insertion request as unrepresentable; the source
  program itself remains valid.
- This experiment establishes lowering for #247. It adds no propagation syntax.
- Toolchain choice: [HOW](how.md). Automated evidence: [PROOF](proof.md).
