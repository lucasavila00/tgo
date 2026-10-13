// Package parser reads Go syntax and every supported TGo form.
//
// Grammar rules read postfix error operators, value-producing if and switch,
// comprehensions, and short returns. They record exact source nodes and spans
// using Go's AST and token positions. The frontend assigns helper names only
// after parsing succeeds. Source call names remain available before the
// frontend creates calls for checking.
//
// Value branches and collection clauses use ordinary statement grammar and
// retain the enclosing source function. Their containers hold syntax for
// checking; the checker interprets their TGo records.
//
// scripts/prepare-frontend.sh applies the tracked patch to the Go source
// selected by internal/checker/patches/go-version. Parser and checker updates
// thus use the same Go version.
package parser
