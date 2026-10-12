package shared

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

// Capture saves addressable operands without reading the destination.
func (r *Context) Capture(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	beforeEvaluation := r.Expression(expr)
	before, values := beforeEvaluation.Statements, beforeEvaluation.Expressions()
	value := values[0]
	if tv := r.Input.Inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
		return before, value
	}
	if len(before) != 0 && r.isTarget(expr) {
		return before, value
	}
	savedEvaluation := r.Save(beforeEvaluation.Values[0])
	saved, refs := savedEvaluation.Statements, savedEvaluation.Expressions()
	return append(before, saved...), refs[0]
}

// place saves the operands of a store without reading its destination.
func (r *Context) place(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	if call, ok := expr.(*ast.CallExpr); ok && r.Input.Comprehensions[call] != nil {
		statementsEvaluation := r.Expression(call)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		return statements, values[0]
	}
	switch e := expr.(type) {
	case *ast.CompositeLit:
		statementsEvaluation := r.Expression(e)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		return statements, values[0]
	case *ast.ParenExpr:
		parenthesized := *e
		before, parenthesized.X = r.place(e.X)
		result = &parenthesized
	case *ast.StarExpr:
		dereference := *e
		before, dereference.X = r.Capture(e.X)
		result = &dereference
	case *ast.IndexExpr:
		index := *e
		kind := storageKinds(r.Input.Inventory.Checked.TypesInfo.TypeOf(e.X))
		if kind == arrayStorage|otherStorage && r.Input.Inventory.Checked.TypesInfo.Types[e.X].Addressable() {
			before, index.X = r.genericBase(e.X)
		} else if kind == arrayStorage {
			before, index.X = r.place(e.X)
		} else {
			before, index.X = r.Capture(e.X)
		}
		indexStatements, indexValue := r.Capture(e.Index)
		before = append(before, indexStatements...)
		index.Index = indexValue
		result = &index
	case *ast.SelectorExpr:
		selector := *e
		if r.Input.Inventory.Checked.TypesInfo.Selections[e] == nil {
			result = &selector
			break
		}
		if _,
			pointer := r.Input.Inventory.Checked.TypesInfo.TypeOf(e.X).Underlying().(*types.Pointer); pointer {
			before, selector.X = r.Capture(e.X)
		} else {
			before, selector.X = r.place(e.X)
		}
		result = &selector
	}
	if expr == r.Input.Target && expr != r.functionState.suppressedRoot {
		r.Request.Inserted = true
		if r.functionState.suppressedRoot != nil {
			savedEvaluation := r.Save(r.ValueOf(r.typedAddress(result, expr)))
			saved, values := savedEvaluation.Statements, savedEvaluation.Expressions()
			before = append(before, saved...)
			result = &ast.StarExpr{X: values[0]}
			before = append(before, r.Input.After([]ast.Expr{result})...)
		} else {
			before = append(before, r.Input.After(nil)...)
		}
	}
	return before, result
}

// address completes address checks before it inserts statements.
func (r *Context) address(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	previous := r.functionState.suppressedRoot
	r.functionState.suppressedRoot = expr
	before, place := r.place(expr)
	r.functionState.suppressedRoot = previous
	savedEvaluation := r.Save(r.ValueOf(r.typedAddress(place, expr)))
	saved, values := savedEvaluation.Statements, savedEvaluation.Expressions()
	before = append(before, saved...)
	if expr == r.Input.Target {
		r.Request.Inserted = true
		before = append(before, r.Input.After([]ast.Expr{&ast.StarExpr{X: values[0]}})...)
	}
	return before, values[0]
}

func (r *Context) typedAddress(value, original ast.Expr) ast.Expr {
	expr := &ast.UnaryExpr{Op: token.AND, X: value}
	r.recordTemporary(expr, types.NewPointer(r.TypeOf(original)), original)
	return expr
}
