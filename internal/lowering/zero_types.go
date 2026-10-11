package lowering

import (
	"go/ast"
	"go/token"
)

// These helpers infer a zero value's type without reading the source storage.
func (r *rewrite) pointerZero(place ast.Expr) ast.Expr {
	if r.zeroPointer == nil {
		r.zeroPointer = r.name()
		parameter, argument, zero := r.name(), r.name(), r.name()
		r.extraDeclarations = append(r.extraDeclarations, zeroDeclaration(r.zeroPointer, parameter, argument, zero, &ast.StarExpr{X: parameter}))
	}
	return &ast.CallExpr{Fun: r.zeroPointer, Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: place}}}
}

func (r *rewrite) sequenceZero(sequence ast.Expr) ast.Expr {
	if r.zeroSequence == nil {
		r.zeroSequence, r.iterAlias = r.name(), r.name()
		r.extraImports = append(r.extraImports, &ast.ImportSpec{Name: r.iterAlias, Path: &ast.BasicLit{Kind: token.STRING, Value: `"iter"`}})
		parameter, argument, zero := r.name(), r.name(), r.name()
		typ := &ast.IndexExpr{X: &ast.SelectorExpr{X: r.iterAlias, Sel: ast.NewIdent("Seq")}, Index: parameter}
		r.extraDeclarations = append(r.extraDeclarations, zeroDeclaration(r.zeroSequence, parameter, argument, zero, typ))
	}
	return &ast.CallExpr{Fun: r.zeroSequence, Args: []ast.Expr{sequence}}
}

func zeroDeclaration(name, parameter, argument, zero *ast.Ident, input ast.Expr) *ast.FuncDecl {
	return &ast.FuncDecl{Name: name, Type: &ast.FuncType{
		TypeParams: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{parameter}, Type: &ast.InterfaceType{Methods: &ast.FieldList{}}}}},
		Params:     &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{argument}, Type: input}}},
		Results:    &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{zero}, Type: parameter}}},
	}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}}
}
