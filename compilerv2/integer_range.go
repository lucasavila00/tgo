package compilerv2

import (
	"go/ast"
	"go/token"
	"go/types"
)

func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

// integerRange keeps the assignment target's type for a constant range count.
func (r *rewrite) integerRange(original, loop *ast.RangeStmt, before []ast.Stmt) []ast.Stmt {
	setup, place := r.place(original.Key)
	before = append(before, setup...)
	var zero ast.Expr
	if index, ok := unparen(place).(*ast.IndexExpr); ok {
		if _, isMap := r.source.Package.TypesInfo.TypeOf(unparen(original.Key).(*ast.IndexExpr).X).Underlying().(*types.Map); isMap {
			r.prepareCollectionImports()
			values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{index.X}}
			zero = r.sequenceZero(values)
		}
	}
	if zero == nil {
		zero = r.pointerZero(place)
	}
	value, first := r.name(), r.name()
	before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{zero}}, &ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.DEFINE, Rhs: []ast.Expr{boolean(true)}})
	repeated := make([]ast.Stmt, len(setup))
	for i, statement := range setup {
		if assign, ok := statement.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
			copy := *assign
			copy.Tok = token.ASSIGN
			repeated[i] = &copy
		} else {
			repeated[i] = statement
		}
	}
	preparation := &ast.IfStmt{Cond: first, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.ASSIGN, Rhs: []ast.Expr{boolean(false)}}}}, Else: &ast.BlockStmt{List: repeated}}
	store := &ast.AssignStmt{Lhs: []ast.Expr{place}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}}
	loop.Key = value
	loop.Value = nil
	loop.Tok = token.ASSIGN
	loop.Body = &ast.BlockStmt{List: []ast.Stmt{preparation, store, loop.Body}}
	return append(before, loop)
}
