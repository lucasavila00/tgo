package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) expression(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if expr == nil {
		return nil, nil
	}
	if !r.contains(expr) {
		return nil, []ast.Expr{expr}
	}
	if statements, values, special := r.specialExpression(expr); special {
		return statements, values
	}
	if parenthesized, ok := expr.(*ast.ParenExpr); ok {
		statements, values := r.expression(parenthesized.X)
		if len(values) != 1 {
			return statements, values
		}
		output := *parenthesized
		output.X = values[0]
		return r.finishExpression(expr, statements, &output)
	}
	statements, result := r.expressionSyntax(expr)
	return r.finishExpression(expr, statements, result)
}

func (r *rewrite) specialExpression(expr ast.Expr) ([]ast.Stmt, []ast.Expr, bool) {
	if operation, ok := r.propagations[expr]; ok && !r.suppressed {
		statements, values := r.propagatedCall(expr.(*ast.CallExpr), operation)
		return statements, values, true
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	if record := r.comprehensions[call]; record != nil {
		statements, values := r.comprehensionCall(call, record)
		return statements, values, true
	}
	if identifier, ok := call.Fun.(*ast.Ident); ok {
		if destination := r.collectionOutputs[identifier.Name]; destination != nil {
			statements, values := r.comprehensionOutput(call, destination)
			return statements, values, true
		}
	}
	if statement := r.conditionals[call]; statement != nil {
		statements, values := r.conditionalCall(call, statement)
		return statements, values, true
	}
	return nil, nil, false
}

func (r *rewrite) expressionSyntax(expr ast.Expr) ([]ast.Stmt, ast.Expr) {
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.FuncLit:
		function := *e
		labels := r.labels
		r.labels = nil
		function.Type, function.Body = r.function(e, e.Type, e.Body)
		r.labels = labels
		result = &function
	case *ast.CallExpr:
		if r.unsafeOperand(e) {
			before, result = r.unevaluatedCall(e)
			break
		}
		call := *e
		operands := append([]ast.Expr{e.Fun}, e.Args...)
		var values []ast.Expr
		before, values = r.operands(operands)
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
		before, values = r.expression(e.X)
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
		before, values = r.expression(e.X)
		dereference.X = values[0]
		result = &dereference
	case *ast.TypeAssertExpr:
		assertion := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		assertion.X = values[0]
		result = &assertion
	case *ast.IndexListExpr:
		indexList := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		indexList.X = values[0]
		result = &indexList
	case *ast.SliceExpr:
		before, result = r.sliceExpression(e)
	case *ast.CompositeLit:
		before, result = r.compositeExpression(e)
	}
	return before, result
}

func (r *rewrite) finishExpression(expr ast.Expr, before []ast.Stmt, result ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if result != expr {
		r.recordOrigin(result, expr)
	}
	if expr == r.target && expr != r.suppressedRoot {
		r.inserted = true
		if r.functionName(expr) {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		if tv := r.source.inventory.Checked.TypesInfo.Types[expr]; tv.Value != nil || tv.IsNil() {
			return append(before, r.after([]ast.Expr{result})...), []ast.Expr{result}
		}
		// Read types from the original expression, not from the new tree.
		saved, refs := r.save(result)
		before = append(before, saved...)
		before = append(before, r.after(refs)...)
		return before, refs
	}
	return before, []ast.Expr{result}
}

func (r *rewrite) functionName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := r.source.inventory.Checked.TypesInfo.Uses[e].(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if r.source.inventory.Checked.TypesInfo.Selections[e] != nil {
			return false
		}
		_, ok := r.source.inventory.Checked.TypesInfo.Uses[e.Sel].(*types.Func)
		return ok
	case *ast.IndexExpr:
		return r.source.inventory.Checked.TypesInfo.Types[e.Index].IsType() && r.functionName(e.X)
	case *ast.IndexListExpr:
		return r.functionName(e.X)
	}
	return false
}

func (r *rewrite) addressableArray(expr ast.Expr) bool {
	array := storageKinds(r.source.inventory.Checked.TypesInfo.TypeOf(expr)) == arrayStorage
	return array && r.source.inventory.Checked.TypesInfo.Types[expr].Addressable()
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
		if obj := r.source.inventory.Checked.TypesInfo.Uses[e]; obj != nil {
			if basic, ok := obj.Type().(*types.Basic); ok {
				return basic.Kind() == types.UntypedBool
			}
		}
	}
	return false
}
