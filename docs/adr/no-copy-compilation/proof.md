# PROOF

These checks are required for the implementation. This ADR does not report
test or benchmark results.

## Behavior and output

Retain existing module, propagation, value-expression, and lowering tests.
Use their handwritten Go references to check execution behavior. Check:

- Mixed Go and TGo packages, nested imports, source-only TGo packages, and
  compilation from a module subdirectory.
- Build constraints, platform and architecture suffixes, internal and external
  test packages, and excluded TGo files with invalid bodies.
- Generated paths with no existing Go file and paths with invalid stale snapshots.
  Stale snapshots must not supply declarations or affect diagnostics.
- Embedded files and package-local cgo headers at their original paths. Check
  supported overlay behavior without restoring a full module copy.
- Relative local replacements, dependency exports, module configuration, and
  a returned module written into another directory.
- Original source positions, generated-name collisions, package identity,
  private type objects, and deterministic output.
- Source checking failures and generated-Go checking failures. Original files,
  snapshots, module files, and directory contents must remain unchanged.
- Original TGo and assets retained in the result, selected generated files
  emitted, excluded generated files absent, and temporary backing files removed.

Run generated programs only after writing returned output. Final overlay checking
must examine generated Go, not the source-checking view.

## Cost

Compare the copy-based and overlay paths on the same toolchain and inputs.
Include several packages, propagation, conditional expressions, and a module
with large unrelated assets. State build-cache and module-cache conditions.

Record elapsed time, allocations, temporary bytes written, and package loads.
Temporary writes must not grow with unrelated asset size. Reads needed to return
those assets remain part of the public API cost. Retain one metadata load and
one final generated-Go load; no load is required for each TGo operator.

Run focused local tests for changed compiler code. Push the implementation
branch and use GitHub Actions for the full compiler suite.

[WHAT](what.md) records the decision. [HOW](how.md) defines the implementation.
