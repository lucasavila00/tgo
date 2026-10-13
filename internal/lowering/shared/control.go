package shared

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/constant"
	"go/token"
	"strconv"
)

// Integer builds an integer literal.
func Integer(value int) ast.Expr { return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)} }

// RangeStatement lowers a range with its original iteration and assignment order.
func (r *Context) RangeStatement(s *ast.RangeStmt) []ast.Stmt {
	exit := r.prepareBreakExit(s, s.X)
	output := *s
	output.Body = r.Block(s.Body)
	statementsEvaluation := r.Expression(s.X)
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	output.X = values[0]
	if s.Tok == token.ASSIGN && (r.Contains(s.Key) || r.Contains(s.Value)) {
		if value := r.input.Inventory.Checked.TypesInfo.Types[s.X].Value; value != nil &&
			value.Kind() == constant.Int {
			return r.withBreakExit(s, &output, r.integerRange(s, &output, statements), exit)
		}
		statements = r.rangeAssignments(s, &output, statements)
	}
	return r.withBreakExit(s, &output, append(statements, &output), exit)
}

// rangeAssignments receives range values in fresh locals and moves destination evaluation into each iteration.
// It updates the copied range so stores occur in source order.
func (r *Context) rangeAssignments(original, output *ast.RangeStmt, statements []ast.Stmt) []ast.Stmt {
	var stores []ast.Stmt
	var destinations []ast.Expr
	var values []ast.Expr
	for _, rangeVariable := range []struct {
		original ast.Expr
		output   *ast.Expr
	}{{original.Key, &output.Key}, {original.Value, &output.Value}} {
		if rangeVariable.original == nil {
			continue
		}
		if name, ok := rangeVariable.original.(*ast.Ident); ok && name.Name == "_" {
			continue
		}
		value := r.Name()
		*rangeVariable.output = value
		statements = append(
			statements,
			Variable(value, r.TemporaryType(r.TypeOf(rangeVariable.original), rangeVariable.original), nil),
		)
		placeStatements, destination := r.place(rangeVariable.original)
		stores = append(stores, placeStatements...)
		destinations = append(destinations, destination)
		values = append(values, value)
	}
	output.Tok = token.ASSIGN
	stores = append(stores, r.orderedStores(destinations, values, token.ASSIGN)...)
	output.Body = &ast.BlockStmt{List: append(stores, output.Body)}
	return statements
}

// switchStatement lowers an ordinary switch without a branch result destination.
func (r *Context) switchStatement(statement *ast.SwitchStmt) []ast.Stmt {
	return r.switchStatementResult(statement, nil, nil)
}

// ValueSwitchStatement lowers a value switch with an explicit branch transformer.
func (r *Context) ValueSwitchStatement(
	statement *ast.SwitchStmt, resultVariable *ast.Ident, transform ValueBlock,
) []ast.Stmt {
	return r.switchStatementResult(statement, resultVariable, transform)
}

// switchStatementResult lowers switch headers and cases, testing case expressions only until a match.
// It uses the supplied branch transformer when the switch produces a value.
func (r *Context) switchStatementResult(
	statement *ast.SwitchStmt,
	resultVariable *ast.Ident,
	transform ValueBlock,
) []ast.Stmt {
	headerNodes := switchHeaderNodes(statement)
	exit := r.prepareBreakExit(statement, headerNodes...)
	rewrittenSwitch := *statement
	var initializerStatements []ast.Stmt
	if statement.Init != nil {
		initializerStatements = r.Header(statement.Init)
	}
	tagEvaluation := r.Expression(statement.Tag)
	tagStatements, tagExpressions := tagEvaluation.Statements, tagEvaluation.Expressions()
	caseClauses, casesNeedStatements := r.switchCases(statement, resultVariable, transform)
	rewrittenSwitch.Body = &ast.BlockStmt{}
	if !casesNeedStatements {
		if len(tagExpressions) != 0 {
			rewrittenSwitch.Tag = tagExpressions[0]
		}
		for _, rewrittenCase := range caseClauses {
			rewrittenSwitch.Body.List = append(rewrittenSwitch.Body.List, rewrittenCase)
		}
		if len(tagStatements) == 0 && len(initializerStatements) <= 1 {
			if len(initializerStatements) != 0 {
				rewrittenSwitch.Init = initializerStatements[0]
			}
			return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{&rewrittenSwitch}, exit)
		}
		rewrittenSwitch.Init = nil
		switchStatements := append(initializerStatements, tagStatements...)
		switchStatements = append(switchStatements, &rewrittenSwitch)
		switchBlock := &ast.BlockStmt{List: switchStatements}
		return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{switchBlock}, exit)
	}
	selectionStatements := append(initializerStatements, tagStatements...)
	var tagValue ast.Expr
	if len(tagExpressions) != 0 {
		tagValue = tagExpressions[0]
		tagInfo := r.input.Inventory.Checked.TypesInfo.Types[statement.Tag]
		if tagInfo.Value == nil {
			savedTagEvaluation := r.Save(r.ValueOf(tagValue))
			savedTagStatements, savedTagValues := savedTagEvaluation.Statements, savedTagEvaluation.Expressions()
			selectionStatements = append(selectionStatements, savedTagStatements...)
			tagValue = savedTagValues[0]
		}
	}
	selectedCase := r.Name()
	selectionStatements = append(
		selectionStatements,
		Variable(selectedCase, r.TemporaryType(types.Typ[types.Int], firstExpression(statement)), Integer(-1)),
	)
	defaultCase := -1
	for i, rewrittenCase := range caseClauses {
		if len(rewrittenCase.List) == 0 {
			defaultCase = i
		}
		selectionStatements = append(
			selectionStatements,
			r.switchCaseSelection(rewrittenCase.List, tagValue, selectedCase, i)...)
		rewrittenCase.List = []ast.Expr{Integer(i)}
		rewrittenSwitch.Body.List = append(rewrittenSwitch.Body.List, rewrittenCase)
	}
	if defaultCase >= 0 {
		selectDefault := &ast.AssignStmt{
			Lhs: []ast.Expr{selectedCase},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{Integer(defaultCase)},
		}
		noCaseSelected := &ast.BinaryExpr{X: selectedCase, Op: token.EQL, Y: Integer(-1)}
		defaultSelection := &ast.IfStmt{Cond: noCaseSelected, Body: &ast.BlockStmt{List: []ast.Stmt{selectDefault}}}
		selectionStatements = append(selectionStatements, defaultSelection)
	}
	rewrittenSwitch.Init = nil
	rewrittenSwitch.Tag = selectedCase
	switchStatements := append(selectionStatements, &rewrittenSwitch)
	switchBlock := &ast.BlockStmt{List: switchStatements}
	return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{switchBlock}, exit)
}

// switchCases copies and lowers case bodies and reports whether case expressions need added statements.
func (r *Context) switchCases(
	statement *ast.SwitchStmt,
	resultVariable *ast.Ident,
	transform ValueBlock,
) ([]*ast.CaseClause, bool) {
	var caseClauses []*ast.CaseClause
	casesNeedStatements := false
	for _, caseStatement := range statement.Body.List {
		originalCase := caseStatement.(*ast.CaseClause)
		for _, caseExpression := range originalCase.List {
			if r.Contains(caseExpression) {
				casesNeedStatements = true
			}
		}
		rewrittenCase := *originalCase
		rewrittenCase.Body = r.switchCaseBody(originalCase, resultVariable, transform)
		caseClauses = append(caseClauses, &rewrittenCase)
	}
	return caseClauses, casesNeedStatements
}

// switchCaseSelection builds ordered case tests that stop evaluating expressions after the first match.
func (r *Context) switchCaseSelection(
	expressions []ast.Expr,
	tagValue ast.Expr,
	selectedCase *ast.Ident,
	caseIndex int,
) []ast.Stmt {
	var selectionStatements []ast.Stmt
	for _, caseExpression := range expressions {
		operandEvaluation := r.Expression(caseExpression)
		operandStatements, operandValues := operandEvaluation.Statements, operandEvaluation.Expressions()
		condition := operandValues[0]
		if tagValue != nil {
			condition = &ast.BinaryExpr{X: tagValue, Op: token.EQL, Y: condition}
		}
		selectCase := &ast.AssignStmt{Lhs: []ast.Expr{selectedCase}, Tok: token.ASSIGN, Rhs: []ast.Expr{Integer(caseIndex)}}
		matchingCase := &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: []ast.Stmt{selectCase}}}
		caseBody := append(operandStatements, matchingCase)
		noCaseSelected := &ast.BinaryExpr{X: selectedCase, Op: token.EQL, Y: Integer(-1)}
		checkCase := &ast.IfStmt{Cond: noCaseSelected, Body: &ast.BlockStmt{List: caseBody}}
		selectionStatements = append(selectionStatements, checkCase)
	}
	return selectionStatements
}

// typeSwitchStatement lowers an ordinary type switch without a branch result destination.
func (r *Context) typeSwitchStatement(statement *ast.TypeSwitchStmt) []ast.Stmt {
	return r.typeSwitchStatementResult(statement, nil, nil)
}

// ValueTypeSwitchStatement lowers a type switch with an explicit branch transformer.
func (r *Context) ValueTypeSwitchStatement(
	statement *ast.TypeSwitchStmt, resultVariable *ast.Ident, transform ValueBlock,
) []ast.Stmt {
	return r.typeSwitchStatementResult(statement, resultVariable, transform)
}

// typeSwitchStatementResult lowers the type-switch header and copies its cases while retaining case bindings.
// It uses the supplied branch transformer when the switch produces a value.
func (r *Context) typeSwitchStatementResult(
	statement *ast.TypeSwitchStmt,
	resultVariable *ast.Ident,
	transform ValueBlock,
) []ast.Stmt {
	exit := r.prepareBreakExit(statement, statement.Init, statement.Assign)
	rewrittenSwitch := *statement
	var headerStatements []ast.Stmt
	if statement.Init != nil {
		headerStatements = r.Header(statement.Init)
	}
	var assertion *ast.TypeAssertExpr
	switch guardStatement := statement.Assign.(type) {
	case *ast.ExprStmt:
		assertion = guardStatement.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		assertion = guardStatement.Rhs[0].(*ast.TypeAssertExpr)
	}
	guardEvaluation := r.Expression(assertion.X)
	guardStatements, guardValues := guardEvaluation.Statements, guardEvaluation.Expressions()
	headerStatements = append(headerStatements, guardStatements...)
	typeAssertion := *assertion
	typeAssertion.X = guardValues[0]
	switch guardStatement := statement.Assign.(type) {
	case *ast.ExprStmt:
		expressionStatement := *guardStatement
		expressionStatement.X = &typeAssertion
		rewrittenSwitch.Assign = &expressionStatement
	case *ast.AssignStmt:
		assignmentStatement := *guardStatement
		assignmentStatement.Rhs = []ast.Expr{&typeAssertion}
		rewrittenSwitch.Assign = &assignmentStatement
	}
	rewrittenSwitch.Body = &ast.BlockStmt{}
	for _, caseStatement := range statement.Body.List {
		originalCase := caseStatement.(*ast.CaseClause)
		rewrittenCase := *originalCase
		rewrittenCase.Body = r.switchCaseBody(originalCase, resultVariable, transform)
		rewrittenSwitch.Body.List = append(rewrittenSwitch.Body.List, &rewrittenCase)
	}
	if len(headerStatements) != 0 {
		rewrittenSwitch.Init = nil
		switchStatements := append(headerStatements, &rewrittenSwitch)
		switchBlock := &ast.BlockStmt{List: switchStatements}
		return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{switchBlock}, exit)
	}
	return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{&rewrittenSwitch}, exit)
}

// switchCaseBody lowers a case body, passing result storage only to cases that supply a value.
func (r *Context) switchCaseBody(clause *ast.CaseClause, resultVariable *ast.Ident, transform ValueBlock) []ast.Stmt {
	if transform != nil {
		branchResult := resultVariable
		if !r.input.Inventory.Checked.TypesInfo.TGoCases[clause] {
			branchResult = nil
		}
		branchBody := &ast.BlockStmt{List: clause.Body}
		return transform(r, branchBody, branchResult).List
	}
	var rewrittenBody []ast.Stmt
	for _, bodyStatement := range clause.Body {
		rewrittenBody = append(rewrittenBody, r.Statement(bodyStatement)...)
	}
	return rewrittenBody
}

// firstExpression finds the first expression in source traversal order for generated-type diagnostics.
func firstExpression(node ast.Node) ast.Expr {
	var result ast.Expr
	ast.Inspect(node, func(n ast.Node) bool {
		if result != nil {
			return false
		}
		if expr, ok := n.(ast.Expr); ok {
			result = expr
			return false
		}
		return true
	})
	return result
}

// switchHeaderNodes collects the initializer, tag, and case expressions whose moved code can contain a break.
func switchHeaderNodes(statement *ast.SwitchStmt) []ast.Node {
	headerNodes := []ast.Node{statement.Init, statement.Tag}
	for _, caseStatement := range statement.Body.List {
		for _, caseExpression := range caseStatement.(*ast.CaseClause).List {
			headerNodes = append(headerNodes, caseExpression)
		}
	}
	return headerNodes
}
