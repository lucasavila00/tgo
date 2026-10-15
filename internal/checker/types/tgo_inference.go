package types

import "github.com/lucasavila00/tgo/internal/ast"

// prepareConditionalArgument records direct and nested branch values for inference.
func (check *Checker) prepareConditionalArgument(expr ast.Expr) {
	call := ast.Unparen(expr)
	if !conditionalExpression(call) {
		return
	}
	if check.tgo.inferConditionals == nil {
		check.tgo.inferConditionals = make(map[ast.Expr]bool)
	}
	check.tgo.inferConditionals[call] = true
	check.prepareConditionalExpression(call)
}

// prepareConditionalTails marks final conditional values for generic inference through nested branches and labels.
func (check *Checker) prepareConditionalTails(statement ast.Stmt) {
	switch statement := statement.(type) {
	case *ast.IfStmt:
		check.prepareConditionalTails(statement.Body)
		if statement.Else != nil {
			check.prepareConditionalTails(statement.Else)
		}
	case *ast.SwitchStmt:
		for _, clause := range statement.Body.List {
			check.prepareConditionalTails(clause)
		}
	case *ast.TypeSwitchStmt:
		for _, clause := range statement.Body.List {
			check.prepareConditionalTails(clause)
		}
	case *ast.CaseClause:
		check.prepareConditionalTails(&ast.BlockStmt{List: statement.Body})
	case *ast.BlockStmt:
		if list := trimTrailingEmptyStmts(statement.List); len(list) != 0 {
			check.prepareConditionalTails(list[len(list)-1])
		}
	case *ast.LabeledStmt:
		check.prepareConditionalTails(statement.Stmt)
	case *ast.ExprStmt:
		check.prepareConditionalArgument(statement.X)
	}
}

// conditionalInference checks each completing branch with the same parameter.
// Terminating branches supply no operand; the other arguments must supply its type.
func (check *Checker) conditionalInference(params *Tuple, args []*operand) (*Tuple, []*operand) {
	var expandedParams []*Var
	var expandedArgs []*operand
	var add func(*Var, *operand)
	add = func(param *Var, arg *operand) {
		if call := ast.Unparen(arg.expr); check.tgo.inferConditionals[call] && arg.isValid() {
			for _, branch := range check.tgo.conditionalValues[call] {
				add(param, branch)
			}
			return
		}
		expandedParams, expandedArgs = append(expandedParams, param), append(expandedArgs, arg)
	}
	for i, arg := range args {
		add(params.At(i), arg)
	}
	return NewTuple(expandedParams...), expandedArgs
}

// assignConditionalArgument checks each branch against the inferred parameter.
func (check *Checker) assignConditionalArgument(arg *operand, typ Type, context string) bool {
	call := ast.Unparen(arg.expr)
	if !check.tgo.inferConditionals[call] || !arg.isValid() {
		return false
	}
	for _, branch := range check.tgo.conditionalValues[call] {
		if !check.assignConditionalArgument(branch, typ, context) {
			check.assignment(branch, typ, context)
		}
		if !branch.isValid() {
			arg.invalidate()
			return true
		}
	}
	arg.mode_, arg.typ_, arg.val = value, typ, nil
	for expr := arg.expr; ; {
		delete(check.untyped, expr)
		check.recordTypeAndValue(expr, value, typ, nil)
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	return true
}

// conditionalExpression reports native expressions with branch values.
func conditionalExpression(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.IfExpr, *ast.SwitchExpr:
		return true
	}
	return false
}

// prepareConditionalExpression marks nested branch values for generic inference.
func (check *Checker) prepareConditionalExpression(expression ast.Expr) {
	switch expression := expression.(type) {
	case *ast.IfExpr:
		check.prepareConditionalTails(expression.Body)
		switch alternative := expression.Else.(type) {
		case *ast.BlockStmt:
			check.prepareConditionalTails(alternative)
		case *ast.IfExpr:
			check.prepareConditionalExpression(alternative)
		}
	case *ast.SwitchExpr:
		check.prepareConditionalTails(expression.Switch)
	}
}
