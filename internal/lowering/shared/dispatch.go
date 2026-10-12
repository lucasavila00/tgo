package shared

import (
	"github.com/lucasavila00/tgo/internal/lowering/source"
	"go/ast"
)

// Dispatch selects rules with the current function context.
type Dispatch struct {
	Expression func(*Context, ast.Expr) Result
	Statement  func(*Context, ast.Stmt) []ast.Stmt
}

// ValueBlock transforms a value switch branch in its current context.
type ValueBlock func(*Context, *ast.BlockStmt, *ast.Ident) *ast.BlockStmt

// Expression returns through dispatch with the current function context.
func (r *Context) Expression(expression ast.Expr) Result {
	return r.Input.Dispatch.Expression(r, expression)
}

// Inventory returns immutable checked source facts.
func (r *Context) Inventory() *source.Inventory { return r.Input.Inventory }

// PropagationAt returns an active operation in the current function.
func (r *Context) PropagationAt(expression ast.Expr) (Propagation, bool) {
	operation, ok := r.Input.Propagations[expression]
	return operation, ok && !r.functionState.suppressed
}
