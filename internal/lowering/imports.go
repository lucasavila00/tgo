package lowering

import (
	"go/ast"
)

func (r *rewrite) slicePackage() *ast.Ident {
	r.prepareSliceImport()
	return r.file.sliceAlias
}
