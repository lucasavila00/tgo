# Parser and scanner preparation

The frontend builds private Go parser and scanner packages from the version in
`../checker/patches/go-version`. The same pin selects the checker source.

Run `make prepare` before building or testing the compiler.
`scripts/prepare-frontend.sh` reads the pinned Go source and applies
`patches/source.patch`. Generated files stay outside Git. The script stops if a
patch requires a shifted match. The package folders retain Go's BSD license.

Grammar rules read every supported TGo form and retain exact source nodes.
The scanner reads postfix operators, applies Go semicolon insertion, and
stores tokens needed for lookahead. The return grammar reads a first or last
`...` result with the required comma. Source spans and line directives refer to the original input.

The parser returns native TGo nodes from the private AST. The checker records
types, lexical scopes, and jump targets on those same nodes. The frontend saves
source call text for error context after parsing succeeds. Value branches and
comprehension clauses retain the enclosing source function.

To update Go, change the shared version pin, review the patch against that
version, and regenerate. Check ordinary Go syntax, TGo grammar, diagnostics,
and fixture snapshots. Handwritten grammar rules are in `parser/tgo_*.go`;
scanner extensions are in `scanner/`. The tracked patch supplies upstream hooks.
