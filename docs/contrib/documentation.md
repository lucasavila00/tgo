# Keep one source for each explanation

Put compiler explanations in package comments, normally `doc.go`. The
[driver](../../internal/driver/readme.md) is the starting point and owns the
shared glossary. Frontend, checker, control-flow, and lowering comments explain
their own work. The [lowering guide](../../internal/lowering/readme.md#follow-the-rewrite)
explains input facts, operand setup, and how to add a lowering rule. Link to
another package instead of repeating its process.

Function comments explain caller behavior: what is read or written, what is
returned, and how failure is reported. Use plain words and small examples.

## Generated package READMEs

Edit Go comments to change a package README. Where local Go work is permitted,
generate and check the result:

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

## Handwritten references

The [user guide](../guide/transpiler.md) owns language syntax and commands; API
details belong to `pkg/transpiler`. [Tests](../../tests/readme.md) owns fixture
use, snapshots, references, and CI. Its README is handwritten because the package
has only test files. The separate fuzz guide covers campaign commands and
failure investigation.

For each new ADR, copy [the template](../adr/template/what.md) into
`docs/adr/<decision>/` with three files:

- `what.md`: the problem, proposed behavior, examples, and language rules.
- `how.md`: the implementation choice, data flow, and reasons for that choice.
- `proof.md`: concrete tests and expected results that verify the behavior.

Use the [HOW template](../adr/template/how.md) and
[PROOF template](../adr/template/proof.md) for the other two files.
Keep each file at most 200 words, including headings and code. Replace all
placeholders and remove sections that do not apply. Use simple English and
small examples. Define necessary terms. Explain decisions rather than listing
code details. Link to package explanations instead of repeating them.
Keep proof about project behavior; execution limits of an agent or machine do
not belong in the decision. Existing ADRs retain their paths until revised.
[Performance](../performance.md) records historical measurements and conditions.
The language vision describes intended features, not only implemented features.

When behavior changes, update its owning comments or reference and any links
that depend on it. Do not add another architecture overview or glossary.
