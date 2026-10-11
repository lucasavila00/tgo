# PROOF

These checks are required for the implementation. This ADR does not report
test or benchmark results.

## Source behavior

Retain existing propagation, module-selection, and expression-lowering tests.
Check:

- Independent and nested calls, multiple success results, and error-only calls.
- Direct error identity, wrapped errors, concrete error types, richer error
  interfaces, type parameters, and incompatible function returns.
- Evaluation order, skipped operands, loops, assignment targets, returns,
  deferred calls, closures, labels, and jumps.
- Ordinary Go files mixed with TGo, imports, build constraints, platform and
  test suffixes, test package variants, and relative module replacements.
- Stale generated files, unchanged original source, source-position diagnostics,
  and deterministic output across repeated compilations.
- Public `Load` package and type-object identity, with private objects kept
  separate.

Use handwritten Go references to compare results, effects, error identity,
panics, and defer order. Follow the
[value-expression proof](../value-expressions/proof.md) for branch types,
context, generic inference, scope, and control flow. Generated Go must pass stock
Go checking. That check does not prove source behavior by itself.

## Checking cost

Measure the old and new paths on the same source and toolchain. Include one
file with increasing counts of independent propagation calls, nested calls,
and calls spread across packages. Record module loads, source package checks,
elapsed time, and allocations. State the dependency-cache conditions.

Require no module reload for each propagation operator and no repeated source
package check for the same package identity in one compilation. Test variants
remain separate packages. Do not set an unmeasured speed target.

## Go updates and validation

Rebuild private upstream source from the pinned version and patches. Check
ordinary Go acceptance and rejection cases as well as TGo feature cases after
a version update. Run focused local checks for changed compiler code. Push the
branch and use GitHub Actions for the full compiler suite.

[WHAT](what.md) records the decision. [HOW](how.md) defines its implementation.
