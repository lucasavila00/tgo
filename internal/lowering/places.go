package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) capture(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	before, values := r.expression(expr)
	value := values[0]
	if tv := r.source.checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
		return before, value
	}
	if len(before) != 0 && r.isTarget(expr) {
		return before, value
	}
	saved, refs := r.save(value)
	return append(before, saved...), refs[0]
}

// place saves the operands of a store without reading its destination.
func (r *rewrite) place(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	if call, ok := expr.(*ast.CallExpr); ok && r.comprehensions[call] != nil {
		before, values := r.expression(call)
		return before, values[0]
	}
	switch e := expr.(type) {
	case *ast.CompositeLit:
		setup, values := r.expression(e)
		return setup, values[0]
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
		kind := storageKinds(r.source.checked.TypesInfo.TypeOf(e.X))
		if kind == arrayStorage|otherStorage && r.source.checked.TypesInfo.Types[e.X].Addressable() {
			before, copy.X = r.genericBase(e.X)
		} else if kind == arrayStorage {
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
		if r.source.checked.TypesInfo.Selections[e] == nil {
			result = &copy
			break
		}
		if _, pointer := r.source.checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer); pointer {
			before, copy.X = r.capture(e.X)
		} else {
			before, copy.X = r.place(e.X)
		}
		result = &copy
	}
	if expr == r.target && expr != r.suppressedRoot {
		r.inserted = true
		if r.suppressedRoot != nil {
			saved, values := r.save(r.typedAddress(result, expr))
			before = append(before, saved...)
			result = &ast.StarExpr{X: values[0]}
			before = append(before, r.after([]ast.Expr{result})...)
		} else {
			before = append(before, r.after(nil)...)
		}
	}
	return before, result
}

// address completes address checks before it inserts statements.
func (r *rewrite) address(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	previous := r.suppressedRoot
	r.suppressedRoot = expr
	before, place := r.place(expr)
	r.suppressedRoot = previous
	saved, values := r.save(r.typedAddress(place, expr))
	before = append(before, saved...)
	if expr == r.target {
		r.inserted = true
		before = append(before, r.after([]ast.Expr{&ast.StarExpr{X: values[0]}})...)
	}
	return before, values[0]
}

func (r *rewrite) typedAddress(value, original ast.Expr) ast.Expr {
	expr := &ast.UnaryExpr{Op: token.AND, X: value}
	if r.temporaryTypes == nil {
		r.temporaryTypes = make(map[ast.Expr]types.Type)
	}
	r.temporaryTypes[expr] = types.NewPointer(r.typeOf(original))
	if r.origins == nil {
		r.origins = make(map[ast.Expr]ast.Expr)
	}
	r.origins[expr] = original
	return expr
}
