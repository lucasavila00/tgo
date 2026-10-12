# Lowering packages: PROOF

## Behavior checks

Use existing propagation, comprehension, value-expression, and insertion tests.
For `use(first(), fetch()!, last())`, require the same trace and original error
as the handwritten Go reference. For skipped Boolean operands and branches,
require no extra execution.

## Rejected inputs

No new rejected inputs or diagnostics. Preserve existing invalid-input results.

## Regression checks

Check tuple and contextual types, comma-ok results, assignment destinations,
store panics, loop post work, labeled jumps, closures, package initializers,
imported types, and error wrapping. Nested functions must retain independent
return targets and jump maps when callbacks receive their current context.

Keep all generated snapshots unchanged. Check imports for dispatcher to rules
to shared, with no shared import of a rule. Check each new package comment for
its implemented responsibility.

## Evidence

Planned: run focused lowering and feature runtime tests, then use hosted CI for
the full compiler suite. Compare snapshots and the public API. Passing checks
must show unchanged behavior and the required import direction.

Completed before the code move: baseline `go test ./internal/lowering` passed.
Package extraction and its behavior checks remain incomplete.
