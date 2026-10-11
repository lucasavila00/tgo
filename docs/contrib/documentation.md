# Generate Go documentation

Edit Go source comments to change the generated package reference. Describe
what each exported function does, its inputs, and its result. Package comments
explain the package's role. Use examples when they help callers.

From the repository root, run:

```sh
make docs
make docs-check
```

`make docs` runs gomarkdoc v1.1.0 and writes
one reference file per package under `docs/reference/`. The configuration is
[.gomarkdoc.yml](../../.gomarkdoc.yml). The package list is in `Makefile`. It
includes the command, public API, and internal packages. Add each new production
package to that list.

Commit the generated file with the source change. `make docs-check` exits with
an error when the file differs from the source. Each package is checked in a
separate tool call. The pull request job runs this check with the test suite.

Templates in `docs/contrib/gomarkdoc/` produce plain headings, signatures,
comments, and examples. Each package has its own file. Unrelated source edits
do not change line links or a repeated symbol list. This keeps review diffs
small and reduces merge conflicts.

Keep package READMEs, guides, architecture documents, and the language vision
as hand-written explanations. The generated reference contains declarations
and their comments. The `tests` package has only test files, so its README
explains the test suite.

The pinned tool runs with `go run`. It does not add a compiler dependency to
`go.mod`. See [gomarkdoc](https://github.com/princjef/gomarkdoc) for its options.
