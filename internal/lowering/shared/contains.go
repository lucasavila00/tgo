package shared

import "go/ast"

// contains reports whether the child's source position range lies inside the parent's range.
func contains(parent, child ast.Node) bool {
	return parent != nil && child != nil && parent.Pos() <= child.Pos() && child.End() <= parent.End()
}
