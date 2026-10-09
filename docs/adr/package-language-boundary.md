# Make the package the language boundary

## Context

TGo now applies language rules by source file. A package can contain handwritten
Go and TGo files, so each tool must preserve two rule sets inside one package.
The package boundary can make these rules smaller and consistent.

## Decision

For each build target, classify a package from handwritten production files
selected by file suffixes and build constraints:

- A TGo package has active production `.tgo` files and no active production
  `.go` files.
- A Go package has active production `.go` files and no active production
  `.tgo` files.
- Every handwritten test must use the language of its production package. A TGo
  package uses `_test.tgo`; a Go package uses `_test.go`.
- Generated Go files, including `*_tgo.go` and `*_tgo_test.go`, do not take part
  in classification and are allowed in a TGo package.

File suffix rules and build constraints select the active files before
classification. Inactive files do not cause a conflict. A package can therefore
have different active files on different targets, but every target must have one
classification.

If both source forms are active, report one package diagnostic before compile,
analysis, formatting, or navigation starts for that package:

```text
package app mixes handwritten TGo and Go files: app.tgo, app_test.go
```

List all active conflicting files in lexical order. Do not emit one diagnostic
for each file.

A Go package can import a TGo package and use its public API. `tgolint` still
loads TGo model facts and validates protected model use in that Go caller. The
boundary removes mixed source from one package; it does not remove cross-package
validation.

Internal and external tests both use the language of the package under test.
Generated output from `model_test.tgo` is `model_tgo_test.go`. It does not
affect classification.

Implement issue #86 first. It migrates the tests that belong to TGo packages to
`_test.tgo`. It does not migrate tests of Go packages. Then implement this
decision and enforce the complete package boundary. This sequence does not make
mixed test sources valid policy.

The compiler bootstrap package remains a Go package. Its implementation and
tests stay in Go so a compiler build does not require an existing TGo compiler.

## Consequences

An existing package with TGo production must move its handwritten Go production
and tests to byte-preserving `.tgo` sources, or split them into a separate Go
package. An existing Go package keeps Go tests. This is the minimum migration
required by the new boundary.

All package-loading tools must use the same active-file classification and the
same mixed-package diagnostic. Go callers keep the current `tgolint` protection
for TGo models.
