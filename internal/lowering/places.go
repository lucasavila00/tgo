package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) capture(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	before, values := r.expression(expr)
	value := values[0]
	if tv := r.source.inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
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
		statements, values := r.expression(call)
		return statements, values[0]
	}
	switch e := expr.(type) {
	case *ast.CompositeLit:
		statements, values := r.expression(e)
		return statements, values[0]
	case *ast.ParenExpr:
		parenthesized := *e
		before, parenthesized.X = r.place(e.X)
		result = &parenthesized
	case *ast.StarExpr:
		dereference := *e
		before, dereference.X = r.capture(e.X)
		result = &dereference
	case *ast.IndexExpr:
		index := *e
		kind := storageKinds(r.source.inventory.Checked.TypesInfo.TypeOf(e.X))
		if kind == arrayStorage|otherStorage && r.source.inventory.Checked.TypesInfo.Types[e.X].Addressable() {
			before, index.X = r.genericBase(e.X)
		} else if kind == arrayStorage {
			before, index.X = r.place(e.X)
		} else {
			before, index.X = r.capture(e.X)
		}
		indexStatements, indexValue := r.capture(e.Index)
		before = append(before, indexStatements...)
		index.Index = indexValue
		result = &index
	case *ast.SelectorExpr:
		selector := *e
		if r.source.inventory.Checked.TypesInfo.Selections[e] == nil {
			result = &selector
			break
		}
		if _, pointer := r.source.inventory.Checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer); pointer {
			before, selector.X = r.capture(e.X)
		} else {
			before, selector.X = r.place(e.X)
		}
		result = &selector
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
	r.recordTemporary(expr, types.NewPointer(r.typeOf(original)), original)
	return expr
}
