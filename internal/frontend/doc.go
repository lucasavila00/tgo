// Package frontend reads TGo syntax into a Go syntax tree.
//
// ParseFile accepts source bytes and returns an AST, or abstract syntax tree,
// with records for propagation and if expressions. Go's standard parser reads
// ordinary headers and statements. The frontend prepares only the added syntax;
// it does not patch the parser.
//
// For postfix propagation, a named comment lets Go parse the surrounding call.
// The frontend then replaces that call with a marker node and records the
// operator, call name, and source position. For if values, it prepares branch
// syntax for parsing and retains the original statements.
//
// A marker identifies a compiler operation. It is not a helper that runs in the
// program. The private checker gives it TGo types and scope rules. The lowerer
// later emits ordinary Go statements. Placeholder functions that carry if syntax
// do not create source return or defer boundaries.
//
// Source offsets map prepared text back to the original input. Independent if
// values can share preparation edits; nested values can require more rounds.
// This package does not type check source or decide where generated statements
// execute.
package frontend
