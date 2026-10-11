# Keep code explanations and API docs in sync

Use architecture docs to explain the complete process. Use package comments to
state each package's job, inputs, and outputs. Function comments explain caller
behavior: what is read or written, what is returned, and how failure is reported.
Define a compiler term before using it and show a small example when it helps.

## Generated package READMEs

Edit Go comments to change a package README. Then generate and check the result:

```sh
make docs
make docs-check
```

`make docs` prepares the private checker and runs gomarkdoc v1.1.0 for each
package in `DOC_PACKAGES` in `Makefile`. `.gomarkdoc.yml` selects the templates
in `docs/contrib/gomarkdoc/`. The output is each package's `readme.md`.

Commit source comments and generated READMEs together. Do not add independent
explanations to a generated README; generation would remove them. The pull
request workflow checks that each README matches its source comments.

The templates show package introductions, signatures, comments, and examples.
They omit changing line links and repeated symbol indexes, so unrelated code
edits do not change generated docs. The pinned tool runs with `go run` and is
not a compiler dependency.

## Handwritten docs

The [architecture guide](../arch/transpiler/readme.md) explains compilation from
start to finish. The user guide explains syntax and commands. Test docs explain
inputs, snapshots, independent references, and CI. Design records explain a
specific decision and its required checks; link to architecture details instead
of repeating the whole process.

The `tests` package has only test files, so its README is handwritten. The
language vision includes intended features and must not be presented as a list
of features already implemented. When behavior changes, update every current-code
explanation that depends on it.
