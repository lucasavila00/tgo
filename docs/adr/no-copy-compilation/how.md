# HOW

## Module views

Use the original module root as the working directory for Go package queries.
Retain Go's module resolution, imports, build constraints, platform suffixes,
test variants, and dependency export data.

Record each TGo input and its absolute generated Go path. The source overlay
presents those paths for package selection, including paths that do not yet
exist. It replaces matching stale snapshots in the build view. Preserve
package declarations, imports, build constraints, and file suffixes in this
view. Metadata preparation does not check TGo feature semantics.

Use [`go/packages` overlays](https://pkg.go.dev/golang.org/x/tools/go/packages#Config)
and the [Go command overlay](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
as required. Unchanged files use their original paths. Do not copy assets,
headers, ordinary source, or dependencies into the overlay directory.

The private loader reads original TGo contents from the recorded input map
and ordinary Go contents from the original paths. Pass TGo through the frontend
to retain propagation, conditional branches, and source positions. Do not check
stub bodies in place of the source. Cache checked packages by package identity.

## Lowering and final checking

Check all selected source packages before lowering. Keep generated Go contents
in memory under their module-relative output paths. Do not write them into the
source directory or read them back from a copied module.

Build the final overlay from those generated contents. It must replace matching
snapshots and hide matching snapshots for excluded TGo inputs. Check this view
with stock Go checking. Keep ordinary Go and non-Go package inputs at their
original locations. Preserve embed and cgo behavior within Go's overlay support.

Use temporary backing files only when the Go tools require them. Remove those
files when compilation ends, including after an error. Go overlays are build
inputs; the caller must still write output before running the generated program.

## Output

Return generated contents directly with original TGo and other module files.
Keep the existing exclusions for `.git`, `.worktrees`, and matching generated
snapshots. Emit generated Go only for selected TGo inputs.

Use the original module files for checking. Keep any replacement-path adjustment
needed by a returned module in output collection, so copied output still refers
to the original local dependencies. Do not edit the original `go.mod` or `go.sum`.

[WHAT](what.md) records the scope. [PROOF](proof.md) defines the required checks.
