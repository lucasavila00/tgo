package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/constant"
	"go/token"
	"strconv"
)

func integer(value int) ast.Expr { return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)} }

func (r *rewrite) rangeStatement(s *ast.RangeStmt) []ast.Stmt {
	exit := r.prepareBreakExit(s, s.X)
	output := *s
	output.Body = r.block(s.Body)
	statements, values := r.expression(s.X)
	output.X = values[0]
	if s.Tok == token.ASSIGN && (r.contains(s.Key) || r.contains(s.Value)) {
		if value := r.input.source.inventory.Checked.TypesInfo.Types[s.X].Value; value != nil && value.Kind() == constant.Int {
			return r.withBreakExit(s, &output, r.integerRange(s, &output, statements), exit)
		}
		statements = r.rangeAssignments(s, &output, statements)
	}
	return r.withBreakExit(s, &output, append(statements, &output), exit)
}

func (r *rewrite) rangeAssignments(original, output *ast.RangeStmt, statements []ast.Stmt) []ast.Stmt {
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
		value := r.name()
		*rangeVariable.output = value
		statements = append(statements, variable(value, r.temporaryType(r.typeOf(rangeVariable.original), rangeVariable.original), nil))
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

func (r *rewrite) switchStatement(statement *ast.SwitchStmt) []ast.Stmt {
	return r.switchStatementResult(statement, nil, false)
}

func (r *rewrite) valueSwitchStatement(statement *ast.SwitchStmt, resultVariable *ast.Ident) []ast.Stmt {
	return r.switchStatementResult(statement, resultVariable, true)
}

func (r *rewrite) switchStatementResult(statement *ast.SwitchStmt, resultVariable *ast.Ident, isValueExpression bool) []ast.Stmt {
	headerNodes := []ast.Node{statement.Init, statement.Tag}
	for _, caseStatement := range statement.Body.List {
		for _, caseExpression := range caseStatement.(*ast.CaseClause).List {
			headerNodes = append(headerNodes, caseExpression)
		}
	}
	exit := r.prepareBreakExit(statement, headerNodes...)
	rewrittenSwitch := *statement
	var initializerStatements []ast.Stmt
	if statement.Init != nil {
		initializerStatements = r.header(statement.Init)
	}
	tagStatements, tagExpressions := r.expression(statement.Tag)
	caseClauses, casesNeedStatements := r.switchCases(statement, resultVariable, isValueExpression)
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
		tagInfo := r.input.source.inventory.Checked.TypesInfo.Types[statement.Tag]
		if tagInfo.Value == nil {
			savedTagStatements, savedTagValues := r.save(tagValue)
			selectionStatements = append(selectionStatements, savedTagStatements...)
			tagValue = savedTagValues[0]
		}
	}
	selectedCase := r.name()
	selectionStatements = append(selectionStatements, variable(selectedCase, r.temporaryType(types.Typ[types.Int], firstExpression(statement)), integer(-1)))
	defaultCase := -1
	for i, rewrittenCase := range caseClauses {
		if len(rewrittenCase.List) == 0 {
			defaultCase = i
		}
		selectionStatements = append(selectionStatements, r.switchCaseSelection(rewrittenCase.List, tagValue, selectedCase, i)...)
		rewrittenCase.List = []ast.Expr{integer(i)}
		rewrittenSwitch.Body.List = append(rewrittenSwitch.Body.List, rewrittenCase)
	}
	if defaultCase >= 0 {
		selectDefault := &ast.AssignStmt{Lhs: []ast.Expr{selectedCase}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(defaultCase)}}
		noCaseSelected := &ast.BinaryExpr{X: selectedCase, Op: token.EQL, Y: integer(-1)}
		defaultSelection := &ast.IfStmt{Cond: noCaseSelected, Body: &ast.BlockStmt{List: []ast.Stmt{selectDefault}}}
		selectionStatements = append(selectionStatements, defaultSelection)
	}
	rewrittenSwitch.Init = nil
	rewrittenSwitch.Tag = selectedCase
	switchStatements := append(selectionStatements, &rewrittenSwitch)
	switchBlock := &ast.BlockStmt{List: switchStatements}
	return r.withBreakExit(statement, &rewrittenSwitch, []ast.Stmt{switchBlock}, exit)
}

func (r *rewrite) switchCases(statement *ast.SwitchStmt, resultVariable *ast.Ident, isValueExpression bool) ([]*ast.CaseClause, bool) {
	var caseClauses []*ast.CaseClause
	casesNeedStatements := false
	for _, caseStatement := range statement.Body.List {
		originalCase := caseStatement.(*ast.CaseClause)
		for _, caseExpression := range originalCase.List {
			if r.contains(caseExpression) {
				casesNeedStatements = true
			}
		}
		rewrittenCase := *originalCase
		rewrittenCase.Body = r.switchCaseBody(originalCase, resultVariable, isValueExpression)
		caseClauses = append(caseClauses, &rewrittenCase)
	}
	return caseClauses, casesNeedStatements
}

func (r *rewrite) switchCaseSelection(expressions []ast.Expr, tagValue ast.Expr, selectedCase *ast.Ident, caseIndex int) []ast.Stmt {
	var selectionStatements []ast.Stmt
	for _, caseExpression := range expressions {
		operandStatements, operandValues := r.expression(caseExpression)
		condition := operandValues[0]
		if tagValue != nil {
			condition = &ast.BinaryExpr{X: tagValue, Op: token.EQL, Y: condition}
		}
		selectCase := &ast.AssignStmt{Lhs: []ast.Expr{selectedCase}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(caseIndex)}}
		matchingCase := &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: []ast.Stmt{selectCase}}}
		caseBody := append(operandStatements, matchingCase)
		noCaseSelected := &ast.BinaryExpr{X: selectedCase, Op: token.EQL, Y: integer(-1)}
		checkCase := &ast.IfStmt{Cond: noCaseSelected, Body: &ast.BlockStmt{List: caseBody}}
		selectionStatements = append(selectionStatements, checkCase)
	}
	return selectionStatements
}

func (r *rewrite) typeSwitchStatement(statement *ast.TypeSwitchStmt) []ast.Stmt {
	return r.typeSwitchStatementResult(statement, nil, false)
}

func (r *rewrite) valueTypeSwitchStatement(statement *ast.TypeSwitchStmt, resultVariable *ast.Ident) []ast.Stmt {
	return r.typeSwitchStatementResult(statement, resultVariable, true)
}

func (r *rewrite) typeSwitchStatementResult(statement *ast.TypeSwitchStmt, resultVariable *ast.Ident, isValueExpression bool) []ast.Stmt {
	exit := r.prepareBreakExit(statement, statement.Init, statement.Assign)
	rewrittenSwitch := *statement
	var headerStatements []ast.Stmt
	if statement.Init != nil {
		headerStatements = r.header(statement.Init)
	}
	var assertion *ast.TypeAssertExpr
	switch guardStatement := statement.Assign.(type) {
	case *ast.ExprStmt:
		assertion = guardStatement.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		assertion = guardStatement.Rhs[0].(*ast.TypeAssertExpr)
	}
	guardStatements, guardValues := r.expression(assertion.X)
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
		rewrittenCase.Body = r.switchCaseBody(originalCase, resultVariable, isValueExpression)
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

func (r *rewrite) switchCaseBody(clause *ast.CaseClause, resultVariable *ast.Ident, isValueExpression bool) []ast.Stmt {
	if isValueExpression {
		branchResult := resultVariable
		if !r.input.source.inventory.Checked.TypesInfo.TGoCases[clause] {
			branchResult = nil
		}
		branchBody := &ast.BlockStmt{List: clause.Body}
		return r.conditionalBlock(branchBody, branchResult).List
	}
	var rewrittenBody []ast.Stmt
	for _, bodyStatement := range clause.Body {
		rewrittenBody = append(rewrittenBody, r.statement(bodyStatement)...)
	}
	return rewrittenBody
}

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
