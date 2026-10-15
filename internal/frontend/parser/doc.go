// Package parser reads Go and TGo with separate grammar modes.
//
// ParseFile reads ordinary Go. ParseTGoFile adds postfix error propagation,
// value-producing if and switch, comprehensions, and shortened returns. The
// loader selects the mode from the input language. The file name does not
// select a grammar.
//
// Each TGo form has a native node in the private AST. Its fields retain the
// written operands and original token positions. Branches and clauses remain
// in their source function. The checker supplies lexical scopes and types;
// AST traversal visits the source children in their written order. ParseTGoFile
// always returns syntax only. The private checker owns identifier bindings
// and lexical scopes through its Info records.
//
// scripts/prepare-frontend.sh applies the tracked patch to the shared pinned
// Go source. The parser, scanner, AST, printer, and checker use that version.
package parser
