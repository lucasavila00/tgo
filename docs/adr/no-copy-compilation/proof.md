# PROOF

Required checks; no results are claimed here.

Retain the [source-checking tests](../source-checking/proof.md) and handwritten
runtime references. Check:

- Mixed and TGo-only packages, subdirectories, imports, build constraints,
  platform suffixes, and internal/external test variants.
- New generated paths, invalid stale snapshots, and excluded invalid inputs.
- Embedded files, package-local cgo headers, local replacements, dependency
  exports, module configuration, and output written elsewhere.
- Positions, name collisions, type identity, deterministic output, retained
  source/assets, and absence of excluded generated files.
- Unchanged original files and directories, including snapshots and module
  files, after source or generated-Go failures. Temporary files are removed.

Final checking must use generated Go. Run programs after writing output.

Compare elapsed time, allocations, temporary bytes, and package loads with the
copy-based implementation. Use identical inputs/toolchains; state cache conditions.
Include multiple packages, propagation, conditionals, and large unrelated assets.
Asset size must not increase temporary writes. Retain one metadata load and one
final Go load. Run focused local tests and the full suite in hosted CI.

[WHAT](what.md) records scope. [HOW](how.md) defines the stages.
