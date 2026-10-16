package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
)

// SlicePackage returns the slices import selected for the current file.
func (r *Context) SlicePackage() *ast.Ident {
	r.PrepareSliceImport()
	return r.file.sliceAlias
}
