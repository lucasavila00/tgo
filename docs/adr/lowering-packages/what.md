# Lowering packages: WHAT

Issue: [#347](https://github.com/lucasavila00/tgo/issues/347)

## Goal

Make each TGo lowering rule easy to find. Keep the dispatcher, rule code, and
shared operations in separate packages. Preserve the public API and generated
Go bytes.

## Example

For `use(first(), fetch()!, last())`, the dispatcher selects propagation.
Shared operand scheduling saves `first()` before the error check. Failure
returns the original error before `last()` or `use` runs.

## Rules

Use this dependency direction:

```text
lowering -> propagation, comprehensions, valueexpr -> shared
```

Shared must not import rule packages. Keep source inventory and name allocation
in their existing separate packages. Keep ordinary Go operations together.

Preserve evaluation order, checked types, assignment order, source scope,
conditional execution, jumps, nested function return targets, and error identity.
A refactor must not reject valid input or change snapshots.

This decision describes planned work. The code move requires the typed result
contract in #336 and the feature and state operations in #346 to reach main.
The package documentation must describe the implemented layout.
