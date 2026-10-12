package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) formatPackage(call ast.Expr) *ast.Ident {
	checked := r.input.source.inventory.Checked
	scope := checked.Types.Scope().Innermost(call.Pos())
	for _, file := range checked.Syntax {
		if !contains(file, call) {
			continue
		}
		for _, spec := range file.Imports {
			object := checked.TypesInfo.Implicits[spec]
			if spec.Name != nil {
				object = checked.TypesInfo.Defs[spec.Name]
			}
			binding, ok := object.(*types.PkgName)
			if !ok || binding.Imported().Path() != "fmt" || binding.Name() == "." || binding.Name() == "_" {
				continue
			}
			_, visible := scope.LookupParent(binding.Name(), call.Pos())
			if visible == binding {
				return ast.NewIdent(binding.Name())
			}
		}
	}
	if r.file.formatAlias == nil {
		r.file.formatAlias = r.nameFor("fmt")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.formatAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`}})
	}
	return r.file.formatAlias
}
