package shared

import (
	"go/ast"
	"go/token"
)

// integerRange keeps the target type and evaluates its address on each iteration.
func (r *Context) integerRange(original, loop *ast.RangeStmt, before []ast.Stmt) []ast.Stmt {
	value := r.Name()
	before = append(before, Variable(value, r.TemporaryType(r.TypeOf(original.Key), original.Key), nil))
	placeStatements, place := r.place(original.Key)
	store := &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}}
	loop.Key = value
	loop.Value = nil
	loop.Tok = token.ASSIGN
	loop.Body = &ast.BlockStmt{List: append(append(placeStatements, store), loop.Body)}
	return append(before, loop)
}
