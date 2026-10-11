package compilerv2

import (
	"go/ast"
	"go/types"
)

func (r *rewrite) capture(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	before, values := r.expression(expr)
	value := values[0]
	if tv := r.source.Package.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
		return before, value
	}
	if len(before) != 0 && expr == r.target {
		return before, value
	}
	saved, refs := r.save(value)
	return append(before, saved...), refs[0]
}

// place saves the operands of a store without reading its destination.
func (r *rewrite) place(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.ParenExpr:
		copy := *e
		before, copy.X = r.place(e.X)
		result = &copy
	case *ast.StarExpr:
		copy := *e
		before, copy.X = r.capture(e.X)
		result = &copy
	case *ast.IndexExpr:
		copy := *e
		baseType := r.source.Package.TypesInfo.TypeOf(e.X).Underlying()
		if _, array := baseType.(*types.Array); array {
			before, copy.X = r.place(e.X)
		} else {
			before, copy.X = r.capture(e.X)
		}
		indexBefore, index := r.capture(e.Index)
		before = append(before, indexBefore...)
		copy.Index = index
		result = &copy
	case *ast.SelectorExpr:
		copy := *e
		if _, pointer := r.source.Package.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer); pointer {
			before, copy.X = r.capture(e.X)
		} else {
			before, copy.X = r.place(e.X)
		}
		result = &copy
	}
	if expr == r.target {
		r.inserted = true
		before = append(before, r.after(nil)...)
	}
	return before, result
}
