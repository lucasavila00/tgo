package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"
)

// sequenceZero builds a call that supplies the sequence element's zero value without iterating the sequence.
// It records the generic helper and iter import once per file.
func (r *Context) sequenceZero(sequence ast.Expr) ast.Expr {
	if r.file.zeroSequence == nil {
		r.file.zeroSequence, r.file.iterAlias = r.Name(), r.NameFor("iter")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.iterAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"iter"`}})
		parameter, argument, zero := r.Name(), r.Name(), r.Name()
		typ := &ast.IndexExpr{X: &ast.SelectorExpr{X: r.file.iterAlias, Sel: ast.NewIdent("Seq")}, Index: parameter}
		r.file.extraDeclarations = append(r.file.extraDeclarations, zeroDeclaration(r.file.zeroSequence,
			parameter, argument, zero, typ))
	}
	return &ast.CallExpr{Fun: r.file.zeroSequence, Args: []ast.Expr{sequence}}
}

// zeroDeclaration builds a generic helper that returns its element type's zero value and leaves its argument unused.
func zeroDeclaration(name, parameter, argument, zero *ast.Ident, input ast.Expr) *ast.FuncDecl {
	return &ast.FuncDecl{Name: name, Type: &ast.FuncType{
		TypeParams: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{parameter},
			Type: &ast.InterfaceType{Methods: &ast.FieldList{}}}}},

		Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{argument}, Type: input}}},
		Results: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{zero}, Type: parameter}}},
	}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}}
}
