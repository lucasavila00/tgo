package shared

import (
	"go/ast"
)

// SlicePackage returns the slices import selected for the current file.
func (r *Context) SlicePackage() *ast.Ident {
	r.PrepareSliceImport()
	return r.file.sliceAlias
}
