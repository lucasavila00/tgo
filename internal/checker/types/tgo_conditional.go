package types

import (
	"go/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

func (check *Checker) conditionalExpr(target *target, x *operand, call *ast.CallExpr, hint Type) exprKind {
	// Branch statements can change state even without a call or receive.
	// Array len, cap, and range must therefore evaluate a conditional value.
	check.hasCallOrRecv = true
	x.expr = call
	x.typ_ = Typ[Invalid]
	body := call.Args[0].(*ast.FuncLit).Body.List[0]
	context := hint
	if target != nil {
		context = target.typ
	}
	values := check.conditionalStatement(body, context)
	if check.tgo.conditionalValues == nil {
		check.tgo.conditionalValues = make(map[*ast.CallExpr][]*operand)
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

func (check *Checker) convertConditional(call *ast.CallExpr, typ Type) {
	check.assignConditionalBranches(check.tgo.conditionalValues[call], typ)
}

func (check *Checker) conditionalBranches(statement *ast.IfStmt, context Type) []*operand {
	previous := check.statementContext
	check.statementContext &^= fallthroughOk | finalSwitchCase | inTypeSwitch
	defer func() { check.statementContext = previous }()
	packageBranch := check.sig == nil && check.scope.Parent() == check.pkg.scope
	check.openScope(statement, "if expression")
	defer check.closeScope()
	if packageBranch {
		defer check.usage(check.scope)
	}
	if statement.Init != nil {
		check.stmt(check.statementContext, statement.Init)
	}
	var condition operand
	check.expr(nil, &condition, statement.Cond)
	if condition.isValid() && !allBoolean(condition.typ()) {
		check.error(statement.Cond, InvalidCond, "if expression condition must be Boolean")
	}
	values := check.conditionalBlock(statement.Body, context)
	switch alternative := statement.Else.(type) {
	case *ast.BlockStmt:
		values = append(values, check.conditionalBlock(alternative, context)...)
	case *ast.IfStmt:
		values = append(values, check.conditionalBranches(alternative, context)...)
	default:
		check.error(statement, InvalidSyntaxTree, "if expression requires else")
		values = append(values, &operand{expr: statement.Cond, typ_: Typ[Invalid]})
	}
	return values
}

func (check *Checker) conditionalBlock(body *ast.BlockStmt, context Type) []*operand {
	check.openScope(body, "conditional expression branch")
	defer check.closeScope()
	return check.conditionalBlockValues(body, context)
}

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
	switch last.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
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

func (check *Checker) conditionalStatement(statement ast.Stmt, context Type) []*operand {
	if statement, ok := statement.(*ast.IfStmt); ok {
		return check.conditionalBranches(statement, context)
	}
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

func (check *Checker) conditionalResultType(call *ast.CallExpr, values []*operand) Type {
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

func (check *Checker) assignConditionalBranches(values []*operand, typ Type) {
	for _, branch := range values {
		check.assignment(branch, typ, "conditional expression branch")
	}
}
