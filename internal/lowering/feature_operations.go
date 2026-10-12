package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// resultStorage declares a typed result and records its source identity.
func (r *rewrite) resultStorage(result *ast.Ident, typ types.Type, original ast.Expr, value ast.Expr) ast.Stmt {
	declaration := variable(result, r.temporaryType(typ, original), value)
	r.recordTemporary(result, typ, original)
	return declaration
}

func (r *rewrite) completeExpression(expression ast.Expr) {
	r.request.lowered[expression] = true
}

func (r *rewrite) completePropagation(expression ast.Expr) {
	r.completeExpression(expression)
	r.request.inserted = true
}
