package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) capture(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	beforeEvaluation := r.expression(expr)
	before, values := beforeEvaluation.statements, beforeEvaluation.expressions()
	value := values[0]
	if tv := r.input.source.inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
		return before, value
	}
	if len(before) != 0 && r.isTarget(expr) {
		return before, value
	}
	savedEvaluation := r.save(beforeEvaluation.values[0])
	saved, refs := savedEvaluation.statements, savedEvaluation.expressions()
	return append(before, saved...), refs[0]
}

// place saves the operands of a store without reading its destination.
func (r *rewrite) place(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	if call, ok := expr.(*ast.CallExpr); ok && r.input.comprehensions[call] != nil {
		statementsEvaluation := r.expression(call)
		statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
		return statements, values[0]
	}
	switch e := expr.(type) {
	case *ast.CompositeLit:
		statementsEvaluation := r.expression(e)
		statements, values := statementsEvaluation.statements, statementsEvaluation.expressions()
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
		kind := storageKinds(r.input.source.inventory.Checked.TypesInfo.TypeOf(e.X))
		if kind == arrayStorage|otherStorage && r.input.source.inventory.Checked.TypesInfo.Types[e.X].Addressable() {
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
		if r.input.source.inventory.Checked.TypesInfo.Selections[e] == nil {
			result = &selector
			break
		}
		if _, pointer := r.input.source.inventory.Checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer); pointer {
			before, selector.X = r.capture(e.X)
		} else {
			before, selector.X = r.place(e.X)
		}
		result = &selector
	}
	if expr == r.input.target && expr != r.functionState.suppressedRoot {
		r.request.inserted = true
		if r.functionState.suppressedRoot != nil {
			savedEvaluation := r.save(r.value(r.typedAddress(result, expr)))
			saved, values := savedEvaluation.statements, savedEvaluation.expressions()
			before = append(before, saved...)
			result = &ast.StarExpr{X: values[0]}
			before = append(before, r.input.after([]ast.Expr{result})...)
		} else {
			before = append(before, r.input.after(nil)...)
		}
	}
	return before, result
}

// address completes address checks before it inserts statements.
func (r *rewrite) address(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	previous := r.functionState.suppressedRoot
	r.functionState.suppressedRoot = expr
	before, place := r.place(expr)
	r.functionState.suppressedRoot = previous
	savedEvaluation := r.save(r.value(r.typedAddress(place, expr)))
	saved, values := savedEvaluation.statements, savedEvaluation.expressions()
	before = append(before, saved...)
	if expr == r.input.target {
		r.request.inserted = true
		before = append(before, r.input.after([]ast.Expr{&ast.StarExpr{X: values[0]}})...)
	}
	return before, values[0]
}

func (r *rewrite) typedAddress(value, original ast.Expr) ast.Expr {
	expr := &ast.UnaryExpr{Op: token.AND, X: value}
	r.recordTemporary(expr, types.NewPointer(r.typeOf(original)), original)
	return expr
}
