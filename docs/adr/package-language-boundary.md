# Make the package the language boundary

## Context

TGo now applies language rules by source file. A package can contain handwritten
Go and TGo files, so each tool must preserve two rule sets inside one package.
The package boundary can make these rules smaller and consistent.

## Decision

Classify each package for the active build context from its handwritten,
non-test source files:

- A TGo package has at least one active `.tgo` file and no active `.go` file.
- A Go package has at least one active `.go` file and no active `.tgo` file.
- Generated `*_tgo.go` files do not take part in classification and are allowed
  in a TGo package.

File suffix rules and build constraints select the active files before
classification. Inactive files do not cause a conflict. A package can therefore
have different active files on different targets, but every target must have one
classification.

If both source forms are active, report one package diagnostic before compile,
analysis, formatting, or navigation starts for that package:

```text
package example.com/app mixes handwritten TGo and Go files: model.tgo, store.go
```

List all active conflicting files in lexical order. Do not emit one diagnostic
for each file.

A Go package can import a TGo package and use its public API. `tgolint` still
loads TGo model facts and validates protected model use in that Go caller. The
boundary removes mixed source from one package; it does not remove cross-package
validation.

Test sources do not classify the non-test package. All Go tests are authored as
`_test.tgo`, as required by issue #86. This applies to internal tests and to
external test packages. Generated test output must end in `_test.go` so that
`go test` recognizes it, and it does not take part in package classification.

Thus, the compiler implementation remains a pure Go package for self-hosting,
while its tests can use TGo. This ADR does not implement test compilation or the
repository migration from issue #86.

## Consequences

Existing mixed packages must move their handwritten Go files to byte-preserving
`.tgo` sources or split them into a separate Go package. This is the minimum
migration required by the new boundary.

All package-loading tools must use the same active-file classification and the
same mixed-package diagnostic. Go callers keep the current `tgolint` protection
for TGo models.
