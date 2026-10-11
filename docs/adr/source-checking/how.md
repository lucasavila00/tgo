# HOW

Use one source-checking path for propagation and value-producing expressions.
Do not retain the repeated helper-checking loop as a second path.

## Compilation stages

1. Find the module and prepare source and generated-Go views as defined by the
   [no-copy compilation decision](../no-copy-compilation/how.md). Original source
   controls compilation; matching generated snapshots cannot affect checking.
2. Use `go/packages` for build selection, test package variants, import metadata,
   and compiled dependency exports. This stage does not check TGo feature rules.
3. Parse selected source with the frontend wrapper and Go's standard parser.
   Keep source positions and operation records with the syntax tree. Marker
   calls identify TGo operations; they are not ordinary source function calls.
4. Check module packages in dependency order with the private checker. Cache
   checked packages by package identity, including test variants. Import compiled
   dependencies with the matching private reader.
5. Pass checked syntax, types, bindings, scopes, and operation records to the
   existing expression lowerer. Finish source checking before building the
   generated-Go view.
6. Check the emitted module with stock Go checking, then return file contents.
   The caller writes the output.

For propagation, check the operand call and its final error result. Validate
the enclosing function's final error interface return and error assignability.
Use the original error type for `!` and `error` for `!!`. Give the propagation
expression its success results, including zero results for an error-only call.
Nested calls then use those results during the same package check.

Keep branch statements in their source function during checking. Parser
placeholders must not create new return, defer, scope, or jump boundaries.
Follow the [value-expression rules](../value-expressions/what.md) and retain
the [expression-lowering behavior](../expression-lowering/what.md).

## Private Go source

Pin the checker and compiled-package reader to the same Go source version.
Store the version, patches, build script, and TGo extensions in Git. Generate
private upstream files during build preparation. A version update must apply
the patches exactly and pass the checks in [PROOF](proof.md).

Keep private type objects separate from stock `go/types` objects. Public `Load`
retains its stock package. If lowering needs private types for that API, check
the same ordinary Go syntax separately with the private checker.

[WHAT](what.md) records the decision and its maintenance cost.
