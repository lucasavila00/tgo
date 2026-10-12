package shared

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// ExpressionSyntax lowers ordinary Go expression syntax and its operands.
func (r *Context) ExpressionSyntax(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.FuncLit:
		function := *e
		function.Type, function.Body = r.function(e, e.Type, e.Body)
		result = &function
	case *ast.CallExpr:
		if r.unsafeOperand(e) {
			before, result = r.unevaluatedCall(e)
			break
		}
		call := *e
		operands := append([]ast.Expr{e.Fun}, e.Args...)
		var values []ast.Expr
		beforeEvaluation := r.Operands(operands)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		call.Fun = values[0]
		call.Args = values[1:]
		result = &call
	case *ast.UnaryExpr:
		unary := *e
		if e.Op == token.AND {
			before, result = r.address(e.X)
			break
		}
		var values []ast.Expr
		beforeEvaluation := r.Expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		unary.X = values[0]
		result = &unary
	case *ast.BinaryExpr:
		before, result = r.binaryExpression(e)
	case *ast.SelectorExpr:
		before, result = r.selectorExpression(e)
	case *ast.IndexExpr:
		before, result = r.indexExpression(e)
	case *ast.StarExpr:
		dereference := *e
		var values []ast.Expr
		beforeEvaluation := r.Expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		dereference.X = values[0]
		result = &dereference
	case *ast.TypeAssertExpr:
		assertion := *e
		var values []ast.Expr
		beforeEvaluation := r.Expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		assertion.X = values[0]
		result = &assertion
	case *ast.IndexListExpr:
		indexList := *e
		var values []ast.Expr
		beforeEvaluation := r.Expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		indexList.X = values[0]
		result = &indexList
	case *ast.SliceExpr:
		before, result = r.sliceExpression(e)
	case *ast.CompositeLit:
		before, result = r.compositeExpression(e)
	}
	return before, result
}

// FinishExpression adds selected insertion work and result storage.
func (r *Context) FinishExpression(expr ast.Expr, before []ast.Stmt, result ast.Expr) Result {
	if result != expr {
		r.recordOrigin(result, expr)
	}
	if expr == r.Input.Target && expr != r.functionState.suppressedRoot {
		r.Request.Inserted = true
		if r.functionName(expr) {
			return r.ResultOf(append(before, r.Input.After([]ast.Expr{result})...), []ast.Expr{result})
		}
		if tv := r.Input.Inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
			return r.ResultOf(append(before, r.Input.After([]ast.Expr{result})...), []ast.Expr{result})
		}
		saved := r.Save(r.ValueOf(result))
		before = append(before, saved.Statements...)
		before = append(before, r.Input.After(saved.Expressions())...)
		saved.Statements = before
		return saved
	}
	return r.ResultOf(before, []ast.Expr{result})
}

func (r *Context) functionName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := r.Input.Inventory.Checked.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.Input.Inventory.Checked.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.Input.Inventory.Checked.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.Input.Inventory.Checked.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *Context) addressableArray(expr ast.Expr) bool {
	array := storageKinds(r.Input.Inventory.Checked.TypesInfo.TypeOf(expr)) == arrayStorage
	return array && r.Input.Inventory.Checked.TypesInfo.Types[expr].Addressable()
}

func (r *Context) arrayBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	return r.place(expr)
}

func (r *Context) untypedBoolean(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return r.untypedBoolean(e.X)
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return true
		case token.LAND, token.LOR:
			return r.untypedBoolean(e.X) && r.untypedBoolean(e.Y)
		}
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return r.untypedBoolean(e.X)
		}
	case *ast.Ident:
		if obj := r.Input.Inventory.Checked.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
