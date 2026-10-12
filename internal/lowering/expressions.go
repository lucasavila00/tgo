package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) expression(expr ast.Expr) expressionResult {
	if expr == nil {
		return r.result(nil, nil)
	}
	if !r.contains(expr) {
		return r.result(nil, []ast.Expr{expr})
	}
	if result, special := r.specialExpression(expr); special {
		return result
	}
	if parenthesized, ok := expr.(*ast.ParenExpr); ok {
		inner := r.expression(parenthesized.X)
		if len(inner.Values) != 1 {
			return inner
		}
		output := *parenthesized
		output.X = inner.Values[0].Expression
		return r.finishExpression(expr, inner.Statements, &output)
	}
	statements, result := r.expressionSyntax(expr)
	return r.finishExpression(expr, statements, result)
}

func (r *rewrite) specialExpression(expr ast.Expr) (expressionResult, bool) {
	if operation, ok := r.input.propagations[expr]; ok && !r.functionState.suppressed {
		return r.propagatedCall(expr.(*ast.CallExpr), operation, false), true
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return expressionResult{}, false
	}
	if record := r.input.comprehensions[call]; record != nil {
		return r.comprehensionCall(call, record), true
	}
	if identifier, ok := call.Fun.(*ast.Ident); ok {
		if destination := r.functionState.collectionOutputs[identifier.Name]; destination != nil {
			return r.comprehensionOutput(call, destination), true
		}
	}
	if statement := r.input.conditionals[call]; statement != nil {
		return r.conditionalCall(call, statement), true
	}
	return expressionResult{}, false
}

func (r *rewrite) expressionSyntax(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
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
		beforeEvaluation := r.operands(operands)
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
		beforeEvaluation := r.expression(e.X)
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
		beforeEvaluation := r.expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		dereference.X = values[0]
		result = &dereference
	case *ast.TypeAssertExpr:
		assertion := *e
		var values []ast.Expr
		beforeEvaluation := r.expression(e.X)
		before, values = beforeEvaluation.Statements, beforeEvaluation.Expressions()
		assertion.X = values[0]
		result = &assertion
	case *ast.IndexListExpr:
		indexList := *e
		var values []ast.Expr
		beforeEvaluation := r.expression(e.X)
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

func (r *rewrite) finishExpression(expr ast.Expr, before []ast.Stmt, result ast.Expr) expressionResult {
	if result != expr {
		r.recordOrigin(result, expr)
	}
	if expr == r.input.target && expr != r.functionState.suppressedRoot {
		r.request.inserted = true
		if r.functionName(expr) {
			return r.result(append(before, r.input.after([]ast.Expr{result})...), []ast.Expr{result})
		}
		if tv := r.input.source.inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
			return r.result(append(before, r.input.after([]ast.Expr{result})...), []ast.Expr{result})
		}
		saved := r.save(r.value(result))
		before = append(before, saved.Statements...)
		before = append(before, r.input.after(saved.Expressions())...)
		saved.Statements = before
		return saved
	}
	return r.result(before, []ast.Expr{result})
}

func (r *rewrite) functionName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := r.input.source.inventory.Checked.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.input.source.inventory.Checked.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.input.source.inventory.Checked.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.input.source.inventory.Checked.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *rewrite) addressableArray(expr ast.Expr) bool {
	array := storageKinds(r.input.source.inventory.Checked.TypesInfo.TypeOf(expr)) == arrayStorage
	return array && r.input.source.inventory.Checked.TypesInfo.Types[expr].Addressable()
}

func (r *rewrite) arrayBase(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	return r.place(expr)
}

func (r *rewrite) untypedBoolean(expr ast.Expr) bool {
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
		if obj := r.input.source.inventory.Checked.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
