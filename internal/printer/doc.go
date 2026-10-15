// Package printer prints ordinary Go nodes from the compiler's private AST.
//
// Lowering replaces TGo syntax before printing. The private printer then uses
// Go's formatting rules with the same node types that lowering produced.
// Comments, source positions, imports, and whitespace keep Go's behavior.
// A separate call-name mode reads source call text for error context. It omits
// propagation operators while it prints the call, such as factory()!().
//
// scripts/prepare-ast.sh builds this package from the shared Go version pin.
// The tracked patch changes the AST import and adds call-name mode. The rules stay
// tied to that pin, together with the parser and checker.
package printer
