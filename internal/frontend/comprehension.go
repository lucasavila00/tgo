package frontend

import (
	"go/ast"
	"go/token"
)

// Comprehension records a typed collection and its range and filter statements.
// Output identifies the leaf call that supplies an element or a key and value.
type Comprehension struct {
	Position token.Position
	Type     ast.Expr
	Body     *ast.BlockStmt
	Output   string
	Ranges   []*ast.RangeStmt
}
