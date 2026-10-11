package frontend

import (
	"go/ast"
	"go/token"
)

// Comprehension records a typed collection, its outputs, and ordered clauses.
// The marker body contains ordinary Go range and filter statements.
type Comprehension struct {
	Position token.Position
	Type     ast.Expr
	Key      ast.Expr
	Value    ast.Expr
	Body     *ast.BlockStmt
}
