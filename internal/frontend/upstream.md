# Parser and scanner preparation

The frontend builds private Go parser and scanner packages from the version in
`../checker/patches/go-version`. The same pin selects the checker source.

Run `make prepare` before building or testing the compiler.
`scripts/prepare-frontend.sh` reads the pinned Go source and applies
`patches/source.patch`. Generated files stay outside Git. The script stops if a
patch requires a shifted match. The package folders retain Go's BSD license.

The scanner reads `!` and adjacent `!!` after a closing parenthesis as postfix
tokens. Each suffix can end a statement. The parser checks the current operand
and records its node and suffix span. This puts operand selection in the grammar.

The frontend publishes records only after parsing succeeds. It saves all source
call names before one walk builds checker calls. Thus a nested helper cannot
appear in another call's error context.

To update Go, change the shared version pin, review the patch against that
version, and regenerate the sources. Check ordinary Go syntax, postfix grammar,
mixed TGo forms, diagnostics, and fixture snapshots. Handwritten parser behavior
is in `parser/tgo.go`; scanner changes remain in the tracked patch.
