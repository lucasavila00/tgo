# PROOF

Run the source-checking tests and handwritten runtime references. Check:

- Data directories, nested modules, and vendor dependencies with invalid TGo
  and snapshots unchanged on disk and retained in output.
- TGo-only and mixed packages, subdirectories, imports, build constraints,
  platform suffixes, and internal/external test variants.
- Invalid stale snapshots, new generated paths, excluded invalid inputs, and
  removal of excluded generated files from disk and returned output.
- Embed assets, cgo headers, vendor selection, local replacements, module
  metadata updates, and output written elsewhere.
- Source positions, fresh names, type identity, repeated compilation, and
  unchanged TGo inputs, ordinary Go, and assets.
- Generated files on disk match returned generated contents. Final checking
  reads generated Go. Correcting an error and recompiling replaces staged files.

Read expected snapshots before compilation. Compile fixture copies so normal
checks leave committed snapshots intact. Compare runtime output with independent
Go references after writing the returned output.

Compare elapsed time and allocations with the copy-based compiler using the
same toolchain and warm caches. Include multiple packages, propagation, if
values, and large unrelated assets. Assets add no compiler write step. Keep one
metadata load and one final Go load.

[WHAT](what.md) records scope. [HOW](how.md) defines the stages.
