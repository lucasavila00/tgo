# Check the source-checking decision

These are required checks, not a report of measured results.

## Types and behavior

Keep propagation, if-value, module-selection, and lowering tests. Cover:

- Independent and nested calls, several success results, and error-only calls.
- Original error identity, wrapping, concrete errors, richer error interfaces,
  type parameters, and incompatible returns.
- If branch types, constants, nil, generic inference, scope, and jumps.
- Evaluation order, skipped operands, loops, assignments, returns, defers,
  closures, and labels.
- Mixed Go and TGo, imports, build constraints, platform and test suffixes,
  local replacements, and stale snapshots.
- Original diagnostic positions and stable repeated output.
- Stock package and type-object identity in public `Load`.

Use handwritten Go references to compare effects, values, errors, panics, and
defer order. Final Go checking must pass, but it cannot prove equivalent
execution by itself. See the [test guide](../../guide/testing.md).

## Checking cost and Go updates

Require no package reload per operator and no repeated source check for the
same package ID in one compilation. Test variants remain separate IDs.

Compare independent calls, nested calls, and several packages on the same
source and toolchain. Record loads, source checks, time, allocations, and cache
conditions. Do not set an unmeasured speed target.

After a Go update, regenerate private source from the pinned version and patch.
Check ordinary Go acceptance and rejection as well as TGo cases. Use hosted CI
for the full compiler suite.

[WHAT](what.md) records the decision. [HOW](how.md) describes its implementation.
