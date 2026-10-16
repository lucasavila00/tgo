package types

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// conditionalExpr checks reachable branch values against a supplied type or finds their common type.
// It records branch operands for later conversion and generic inference.
func (check *Checker) conditionalExpr(target *target, x *operand, call ast.Expr, hint Type) exprKind {
	defer check.recordRuntimeExpression()
	// Branch statements can change state even without a call or receive.
	// Array len, cap, and range must therefore evaluate a conditional value.
	x.expr = call
	x.typ_ = Typ[Invalid]
	body := call
	context := hint
	if target != nil {
		context = target.typ
	}
	values := check.conditionalValue(body, context)
	if check.tgo.conditionalValues == nil {
		check.tgo.conditionalValues = make(map[ast.Expr][]*operand)
	}
	check.tgo.conditionalValues[call] = values
	for _, branch := range values {
		if !branch.isValid() {
			return expression
		}
	}
	// Generic calls determine the parameter type from all branch values.
	if context == nil && check.tgo.inferConditionals[call] {
		typ := Type(Typ[UntypedNil])
		if len(values) != 0 {
			typ = values[0].typ()
		}
		x.mode_, x.typ_ = value, typ
		return expression
	}
	context = check.conditionalContext(context, values)
	if context != nil {
		for _, branch := range values {
			check.assignment(branch, context, "conditional expression branch")
			if !branch.isValid() {
				return expression
			}
		}
		x.mode_, x.typ_ = value, context
		return expression
	}
	typ := check.conditionalResultType(call, values)
	if typ == nil {
		return expression
	}
	x.mode_, x.typ_ = value, typ
	return expression
}

// conditionalContext keeps string branches as strings when copy or append permits a byte-slice or string source.
func (check *Checker) conditionalContext(context Type, values []*operand) Type {
	// A byte-slice source of copy or append can instead be a string.
	// Keep its own string type when every value branch produces a string.
	if context != nil && check.tgo.sliceSourceContext != nil && Identical(context, check.tgo.sliceSourceContext) &&
		len(values) != 0 {
		strings := true
		for _, branch := range values {
			strings = strings && allString(branch.typ())
		}
		if strings {
			context = nil
		}
	}
	return context
}

// convertConditional assigns a chosen type to the saved branch operands.
func (check *Checker) convertConditional(call ast.Expr, typ Type) {
	check.assignConditionalBranches(check.tgo.conditionalValues[call], typ)
}

// conditionalBranches checks the initializer, Boolean condition, and both branches in an if-expression scope.
func (check *Checker) conditionalBranches(statement *ast.IfExpr, context Type) []*operand {
	check.recordExpressionScope(statement)
	return check.ifBranchValues(statement, statement.Init, statement.Cond, statement.Body, statement.Else, context)
}

// statementBranchValues checks an ordinary if used as a branch's final value.
func (check *Checker) statementBranchValues(statement *ast.IfStmt, context Type) []*operand {
	return check.ifBranchValues(statement, statement.Init, statement.Cond, statement.Body, statement.Else, context)
}

// ifBranchValues checks the source scope, condition, and alternatives.
func (check *Checker) ifBranchValues(statement ast.Node, init ast.Stmt, cond ast.Expr,
	body *ast.BlockStmt, alternative ast.Node, context Type) []*operand {
	previous := check.statementContext
	check.statementContext &^= fallthroughOk | finalSwitchCase | inTypeSwitch
	defer func() { check.statementContext = previous }()
	packageBranch := check.sig == nil && check.scope.Parent() == check.pkg.scope
	check.openScope(statement, "if expression")
	defer check.closeScope()
	if packageBranch {
		check.queuePackageUsage(check.scope)
	}
	if init != nil {
		check.stmt(check.statementContext, init)
	}
	var condition operand
	check.expr(nil, &condition, cond)
	if condition.isValid() && !allBoolean(condition.typ()) {
		check.error(cond, InvalidCond, "if expression condition must be Boolean")
	}
	values := check.conditionalBlock(body, context)
	switch alternative := alternative.(type) {
	case *ast.BlockStmt:
		values = append(values, check.conditionalBlock(alternative, context)...)
	case *ast.IfExpr:
		values = append(values, check.conditionalBranches(alternative, context)...)
	case *ast.IfStmt:
		values = append(values, check.statementBranchValues(alternative, context)...)
	default:
		check.error(statement, InvalidSyntaxTree, "if expression requires else")
		values = append(values, &operand{expr: cond, typ_: Typ[Invalid]})
	}
	return values
}

// conditionalBlock opens a branch scope and checks its statements and final value.
func (check *Checker) conditionalBlock(body *ast.BlockStmt, context Type) []*operand {
	check.openScope(body, "conditional expression branch")
	defer check.closeScope()
	return check.conditionalBlockValues(body, context)
}

// conditionalBlockValues checks branch statements and the final value.
// It records whether the branch can finish and excludes values on paths that cannot finish.
func (check *Checker) conditionalBlockValues(body *ast.BlockStmt, context Type) []*operand {
	list := trimTrailingEmptyStmts(body.List)
	if len(list) == 0 {
		check.error(body, InvalidSyntaxTree, "conditional expression branch requires a final value")
		return []*operand{{expr: &ast.BadExpr{From: body.Pos()}, typ_: Typ[Invalid]}}
	}
	check.stmtList(check.statementContext, list[:len(list)-1])
	flow := check.branchFlow(body)
	if !flow.Tail {
		context = nil
	}
	values := check.conditionalTail(list[len(list)-1], context, flow.Tail)
	flow = check.branchFlow(body)
	if check.Info != nil {
		if check.TGoBranches == nil {
			check.TGoBranches = make(map[*ast.BlockStmt]bool)
		}
		check.TGoBranches[body] = flow.End
	}
	if !flow.End {
		return nil
	}
	return values
}

// conditionalTail checks a branch final statement as a value, nested conditional, or control-flow exit.
func (check *Checker) conditionalTail(original ast.Stmt, context Type, reachable bool) []*operand {
	last := original
	for {
		labelled, ok := last.(*ast.LabeledStmt)
		if !ok {
			break
		}
		check.hasLabel = true
		last = labelled.Stmt
	}
	switch last := last.(type) {
	case *ast.IfStmt:
		return check.statementBranchValues(last, context)
	case *ast.SwitchStmt, *ast.TypeSwitchStmt:
		return check.conditionalStatement(last, context)
	}
	if statement, ok := last.(*ast.ExprStmt); ok {
		branch := new(operand)
		check.rawExpr(newTarget(context, "conditional expression branch"), branch, statement.X, nil, false)
		check.exclude(branch, 1<<builtin|1<<typexpr)
		if branch.mode() == novalue && check.isTerminating(statement, "") {
			return nil
		}
		check.singleValue(branch)
		if branch.mode() == novalue && reachable {
			check.error(statement, InvalidSyntaxTree, "conditional expression branch requires one value")
			branch.invalidate()
		}
		return []*operand{branch}
	}
	check.stmt(check.statementContext, original)
	if check.isTerminating(original, "") || !reachable {
		return nil
	}
	if branch, ok := last.(*ast.BranchStmt); ok && (branch.Tok == token.BREAK || branch.Tok == token.CONTINUE) {
		return nil
	}
	check.error(last, InvalidSyntaxTree, "conditional expression branch requires a final value")
	return []*operand{{expr: &ast.BadExpr{From: last.Pos()}, typ_: Typ[Invalid]}}
}

type conditionalCase struct {
	context Type
	values  []*operand
}

// conditionalStatement checks an if or switch value and collects its branch operands.
// For switches, it requires a default and prevents break from targeting the value switch.
func (check *Checker) conditionalStatement(statement ast.Stmt, context Type) []*operand {
	var switchBody *ast.BlockStmt
	switch statement := statement.(type) {
	case *ast.SwitchStmt:
		switchBody = statement.Body
	case *ast.TypeSwitchStmt:
		switchBody = statement.Body
	}
	if check.tgo.conditionalCases == nil {
		check.tgo.conditionalCases = make(map[*ast.CaseClause]*conditionalCase)
	}
	hasDefaultCase := false
	for _, caseStatement := range switchBody.List {
		clause := caseStatement.(*ast.CaseClause)
		hasDefaultCase = hasDefaultCase || clause.List == nil
		check.tgo.conditionalCases[clause] = &conditionalCase{context: context}
	}
	if !hasDefaultCase {
		check.error(statement, InvalidSyntaxTree, "switch expression requires default")
	}
	// The first visited switch is the value switch, not a permitted break target.
	ast.Walk(valueSwitchBreakVisitor{check: check, nestedBreakTargets: -1}, statement)
	check.stmt(check.statementContext, statement)
	var branchValues []*operand
	for _, caseStatement := range switchBody.List {
		clause := caseStatement.(*ast.CaseClause)
		branchValues = append(branchValues, check.tgo.conditionalCases[clause].values...)
		delete(check.tgo.conditionalCases, clause)
	}
	if !hasDefaultCase {
		invalidBranch := &operand{expr: &ast.BadExpr{From: switchBody.Pos()}, typ_: Typ[Invalid]}
		branchValues = append(branchValues, invalidBranch)
	}
	return branchValues
}

// conditionalCase checks one value-switch case with its supplied type and records whether it can finish.
func (check *Checker) conditionalCase(clause *ast.CaseClause, context Type) []*operand {
	enclosingContext := check.statementContext
	check.statementContext &^= breakOk | fallthroughOk | finalSwitchCase
	defer func() { check.statementContext = enclosingContext }()
	caseBody := &ast.BlockStmt{Lbrace: clause.Colon, List: clause.Body, Rbrace: clause.End()}
	branchValues := check.conditionalBlockValues(caseBody, context)
	if check.Info != nil {
		if check.TGoCases == nil {
			check.TGoCases = make(map[*ast.CaseClause]bool)
		}
		check.TGoCases[clause] = check.TGoBranches[caseBody]
	}
	return branchValues
}

// conditionalResultType finds a common branch type and checks assignments to it.
// It requires an annotation when typed branches conflict or no result type is available.
func (check *Checker) conditionalResultType(call ast.Expr, values []*operand) Type {
	var typ Type
	for _, branch := range values {
		candidate := branch.typ()
		if isUntyped(candidate) {
			continue
		}
		if typ != nil && !Identical(typ, candidate) {
			check.error(call, IncompatibleAssign, "conditional branch types differ; add an explicit type annotation")
			return nil
		}
		typ = candidate
	}
	if typ != nil {
		check.assignConditionalBranches(values, typ)
		for _, branch := range values {
			if !branch.isValid() {
				return nil
			}
		}
	} else {
		typ = untypedConditionalType(values)
	}
	if typ == nil {
		check.error(call, IncompatibleAssign, "conditional result type is unknown; add an explicit type annotation")
		return nil
	}
	return typ
}

// assignConditionalBranches checks and converts each branch operand for assignment to the supplied type.
func (check *Checker) assignConditionalBranches(values []*operand, typ Type) {
	for _, branch := range values {
		check.assignment(branch, typ, "conditional expression branch")
	}
}

// conditionalValue checks the branches of a native value expression.
func (check *Checker) conditionalValue(expression ast.Expr, context Type) []*operand {
	switch expression := expression.(type) {
	case *ast.IfExpr:
		return check.conditionalBranches(expression, context)
	case *ast.SwitchExpr:
		return check.conditionalStatement(expression.Switch, context)
	}
	panic("invalid conditional expression")
}
