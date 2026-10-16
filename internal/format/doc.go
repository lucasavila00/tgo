// Package format formats ordinary Go from the compiler's private AST.
//
// The formatter uses the shared pinned Go printer and parses ordinary Go when
// it sorts imports or formats a source fragment. Lowering supplies Go nodes;
// TGo source forms are handled before this output step.
//
// scripts/prepare-ast.sh builds the package with the same source version as
// the AST, parser, and checker. The tracked patch changes their import paths.
package format
