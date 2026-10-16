package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Value keeps a Go expression with its checked type and source identity.
// A tuple type means that the expression supplies multiple results.
type Value struct {
	Expression ast.Expr
	Type       types.Type
	Original   ast.Expr
	Temporary  bool
}

// Result keeps evaluation statements with the values they produce.
type Result struct {
	Statements []ast.Stmt
	Values     []Value
}

// Expressions returns the Go nodes for statement construction.
func (result Result) Expressions() []ast.Expr {
	if result.Values == nil {
		return nil
	}
	expressions := make([]ast.Expr, len(result.Values))
	for i, value := range result.Values {
		expressions[i] = value.Expression
	}
	return expressions
}
