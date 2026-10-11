package lowering

import (
	"go/ast"
	"go/token"
	"go/types"
)

// integerRange keeps the assignment target's type for a constant range count.
func (r *rewrite) integerRange(original, loop *ast.RangeStmt, before []ast.Stmt) []ast.Stmt {
	setup, place := r.place(original.Key)
	before = append(before, setup...)
	var zero ast.Expr
	if index, ok := ast.Unparen(place).(*ast.IndexExpr); ok {
		if _, isMap := r.source.Package.TypesInfo.TypeOf(ast.Unparen(original.Key).(*ast.IndexExpr).X).Underlying().(*types.Map); isMap {
			r.prepareCollectionImports()
			values := &ast.CallExpr{Fun: &ast.SelectorExpr{X: r.mapAlias, Sel: ast.NewIdent("Values")}, Args: []ast.Expr{index.X}}
			zero = r.sequenceZero(values)
		}
	}
	if zero == nil {
		zero = r.pointerZero(place)
	}
	value, first := r.name(), r.name()
	before = append(before, variable(value, r.temporaryType(r.typeOf(original.Key), original.Key), zero), variable(first, r.temporaryType(types.Typ[types.Bool], original.Key), boolean(true)))
	var repeated []ast.Stmt
	for _, statement := range setup {
		if declaration, ok := statement.(*ast.DeclStmt); ok {
			spec := declaration.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
			if len(spec.Values) != 0 {
				repeated = append(repeated, &ast.AssignStmt{Lhs: []ast.Expr{spec.Names[0]}, Tok: token.ASSIGN, Rhs: spec.Values})
			}
		} else {
			repeated = append(repeated, statement)
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
