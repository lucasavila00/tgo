package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// ResultStorage declares a typed result and records its source identity.
func (r *Context) ResultStorage(result *ast.Ident, typ types.Type, original ast.Expr, value ast.Expr) ast.Stmt {
	declaration := Variable(result, r.TemporaryType(typ, original), value)
	r.recordTemporary(result, typ, original)
	return declaration
}

// CompleteExpression records completion of a selected source expression.
func (r *Context) CompleteExpression(expression ast.Expr) {
	r.request.lowered[expression] = true
}

// CompletePropagation records a completed error propagation and its insertion.
func (r *Context) CompletePropagation(expression ast.Expr) {
	r.CompleteExpression(expression)
	r.request.inserted = true
}
