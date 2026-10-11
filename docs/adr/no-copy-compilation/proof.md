# Check compilation in place

Cover these cases through the public compiler API:

- TGo-only and mixed packages, imports, subdirectory arguments, build tags,
  platform suffixes, and test variants.
- Invalid stale snapshots and excluded inputs. Owned selected outputs must be
  replaced; excluded outputs must be removed.
- Invalid TGo and snapshots in data directories, nested modules, and vendor
  dependencies. These must remain unchanged.
- Embedded assets, cgo headers, vendor and workspace dependencies, relative
  replacements, and module metadata updates.
- Original diagnostic positions, fresh names, type identity, repeated
  compilation, and unchanged inputs and assets.
- A source-checking failure followed by a correction. Recompilation must
  replace the failed stage and final stock Go checking must pass.

Read committed snapshots before compiling test fixture copies. Compare the
files written on disk with those snapshots. Run the compiled fixtures and
handwritten Go references to compare behavior.

Measure time and allocations with the same toolchain and warm caches. Include
several packages, propagation, if values, and large unrelated assets. The driver
must not read unrelated asset contents or write assets. Keep one metadata load
and one final Go load.

[WHAT](what.md) records the decision. [HOW](how.md) describes the implementation.
