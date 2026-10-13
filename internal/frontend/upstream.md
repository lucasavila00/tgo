# Parser and scanner preparation

The frontend builds private Go parser and scanner packages from the version in
`../checker/patches/go-version`. The same pin selects the checker source.

Run `make prepare` before building or testing the compiler.
`scripts/prepare-frontend.sh` reads the pinned Go source and applies
`patches/source.patch`. Generated files stay outside Git. The script stops if a
patch requires a shifted match. The package folders retain Go's BSD license.

Grammar rules read every supported TGo form and retain exact source nodes.
The scanner reads postfix and short-return operators, applies semicolon
insertion, and stores tokens needed for lookahead. Source spans and line
directives refer to the original input.

The frontend publishes records only after parsing succeeds. It saves source
call names before naming syntax containers and creating checker calls. Value
branches and comprehension clauses retain the enclosing source function.

To update Go, change the shared version pin, review the patch against that
version, and regenerate. Check ordinary Go syntax, TGo grammar, diagnostics,
and fixture snapshots. Handwritten grammar rules are in `parser/tgo_*.go`;
scanner extensions are in `scanner/`. The tracked patch supplies upstream hooks.
