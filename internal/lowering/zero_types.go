package lowering

import (
	"go/ast"
	"go/token"
)

func (r *rewrite) sequenceZero(sequence ast.Expr) ast.Expr {
	if r.file.zeroSequence == nil {
		r.file.zeroSequence, r.file.iterAlias = r.name(), r.nameFor("iter")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.iterAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"iter"`}})
		parameter, argument, zero := r.name(), r.name(), r.name()
		typ := &ast.IndexExpr{X: &ast.SelectorExpr{X: r.file.iterAlias, Sel: ast.NewIdent("Seq")}, Index: parameter}
		r.file.extraDeclarations = append(r.file.extraDeclarations, zeroDeclaration(r.file.zeroSequence,
			parameter, argument, zero, typ))
	}
	return &ast.CallExpr{Fun: r.file.zeroSequence, Args: []ast.Expr{sequence}}
}

func zeroDeclaration(name, parameter, argument, zero *ast.Ident, input ast.Expr) *ast.FuncDecl {
	return &ast.FuncDecl{Name: name, Type: &ast.FuncType{
		TypeParams: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{parameter},
			Type: &ast.InterfaceType{Methods: &ast.FieldList{}}}}},

		Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{argument}, Type: input}}},
		Results: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{zero}, Type: parameter}}},
	}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}}
}
