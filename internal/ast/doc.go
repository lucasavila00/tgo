// Package ast holds Go and TGo source syntax in one tree.
//
// The parser gives each TGo form its own node. Its fields retain source
// operands, branches, clauses, and token positions. The checker records types
// and bindings on those nodes. Lowering reads the same nodes and emits ordinary
// Go syntax. This preserves node identity between the compiler passes.
//
// Value branches and comprehension clauses belong to their source function.
// A function literal represents a function written in the source. It does not
// serve as a container for another language form.
//
// scripts/prepare-ast.sh builds the Go node definitions and traversal from the
// shared Go version pin. Handwritten definitions add TGo syntax in this package,
// where they can implement its expression and statement interfaces directly.
package ast
