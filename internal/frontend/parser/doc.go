// Package parser reads Go syntax and TGo postfix error operators.
//
// The primary-expression rule attaches ! or !! to the call just parsed.
// A callback records the exact operand and suffix span.
// This keeps operand selection in the grammar and lets the frontend own
// source records, names, and checker-call construction. Syntax uses Go's AST
// and token positions.
//
// scripts/prepare-frontend.sh applies the tracked patch to the Go source
// selected by internal/checker/patches/go-version. Parser and checker updates
// thus use the same Go version.
package parser
