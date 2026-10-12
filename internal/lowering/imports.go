package lowering

import (
	"go/ast"
	"go/token"
)

func (r *rewrite) formatPackage() *ast.Ident {
	if r.file.formatAlias == nil {
		r.file.formatAlias = r.nameFor("fmt")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.formatAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`}})
	}
	return r.file.formatAlias
}

func (r *rewrite) slicePackage() *ast.Ident {
	r.prepareSliceImport()
	return r.file.sliceAlias
}
