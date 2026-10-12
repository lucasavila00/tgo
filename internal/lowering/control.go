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
		if value := r.source.inventory.Checked.TypesInfo.Types[s.X].Value; value != nil && value.Kind() == constant.Int {
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

func (r *rewrite) switchStatement(s *ast.SwitchStmt) []ast.Stmt {
	roots := []ast.Node{s.Init, s.Tag}
	for _, stmt := range s.Body.List {
		for _, expr := range stmt.(*ast.CaseClause).List {
			roots = append(roots, expr)
		}
	}
	exit := r.prepareBreakExit(s, roots...)
	output := *s
	var initStatements []ast.Stmt
	if s.Init != nil {
		initStatements = r.header(s.Init)
	}
	tagStatements, tagValues := r.expression(s.Tag)
	cases, changedCases := r.switchCases(s)
	output.Body = &ast.BlockStmt{}
	if !changedCases {
		if len(tagValues) != 0 {
			output.Tag = tagValues[0]
		}
		for _, clause := range cases {
			output.Body.List = append(output.Body.List, clause)
		}
		if len(tagStatements) == 0 && len(initStatements) <= 1 {
			if len(initStatements) != 0 {
				output.Init = initStatements[0]
			}
			return r.withBreakExit(s, &output, []ast.Stmt{&output}, exit)
		}
		output.Init = nil
		statements := append(initStatements, tagStatements...)
		return r.withBreakExit(s, &output, []ast.Stmt{&ast.BlockStmt{List: append(statements, &output)}}, exit)
	}
	statements := append(initStatements, tagStatements...)
	var tagValue ast.Expr
	if len(tagValues) != 0 {
		tagValue = tagValues[0]
		if tv := r.source.inventory.Checked.TypesInfo.Types[s.Tag]; tv.Value == nil {
			saveStatements, values := r.save(tagValue)
			statements = append(statements, saveStatements...)
			tagValue = values[0]
		}
	}
	selected := r.name()
	statements = append(statements, variable(selected, r.temporaryType(types.Typ[types.Int], firstExpression(s)), integer(-1)))
	defaultCase := -1
	for i, clause := range cases {
		if len(clause.List) == 0 {
			defaultCase = i
		}
		statements = append(statements, r.switchCaseSelection(clause.List, tagValue, selected, i)...)
		clause.List = []ast.Expr{integer(i)}
		output.Body.List = append(output.Body.List, clause)
	}
	if defaultCase >= 0 {
		selectDefault := &ast.AssignStmt{Lhs: []ast.Expr{selected}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(defaultCase)}}
		statements = append(statements, &ast.IfStmt{Cond: &ast.BinaryExpr{X: selected, Op: token.EQL, Y: integer(-1)}, Body: &ast.BlockStmt{List: []ast.Stmt{selectDefault}}})
	}
	output.Init = nil
	output.Tag = selected
	return r.withBreakExit(s, &output, []ast.Stmt{&ast.BlockStmt{List: append(statements, &output)}}, exit)
}

func (r *rewrite) switchCases(s *ast.SwitchStmt) ([]*ast.CaseClause, bool) {
	var cases []*ast.CaseClause
	changed := false
	for _, statement := range s.Body.List {
		original := statement.(*ast.CaseClause)
		for _, expression := range original.List {
			if r.contains(expression) {
				changed = true
			}
		}
		clause := *original
		clause.Body = nil
		for _, statement := range original.Body {
			clause.Body = append(clause.Body, r.statement(statement)...)
		}
		cases = append(cases, &clause)
	}
	return cases, changed
}

func (r *rewrite) switchCaseSelection(expressions []ast.Expr, tagValue ast.Expr, selected *ast.Ident, caseIndex int) []ast.Stmt {
	var statements []ast.Stmt
	for _, expression := range expressions {
		expressionStatements, values := r.expression(expression)
		condition := values[0]
		if tagValue != nil {
			condition = &ast.BinaryExpr{X: tagValue, Op: token.EQL, Y: condition}
		}
		selectCase := &ast.AssignStmt{Lhs: []ast.Expr{selected}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(caseIndex)}}
		caseBody := append(expressionStatements, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: []ast.Stmt{selectCase}}})
		caseNotSelected := &ast.BinaryExpr{X: selected, Op: token.EQL, Y: integer(-1)}
		statements = append(statements, &ast.IfStmt{Cond: caseNotSelected, Body: &ast.BlockStmt{List: caseBody}})
	}
	return statements
}

func (r *rewrite) typeSwitchStatement(s *ast.TypeSwitchStmt) []ast.Stmt {
	exit := r.prepareBreakExit(s, s.Init, s.Assign)
	output := *s
	var statements []ast.Stmt
	if s.Init != nil {
		statements = r.header(s.Init)
	}
	var assertion *ast.TypeAssertExpr
	switch assign := s.Assign.(type) {
	case *ast.ExprStmt:
		assertion = assign.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		assertion = assign.Rhs[0].(*ast.TypeAssertExpr)
	}
	assertionStatements, values := r.expression(assertion.X)
	statements = append(statements, assertionStatements...)
	typeAssertion := *assertion
	typeAssertion.X = values[0]
	switch assign := s.Assign.(type) {
	case *ast.ExprStmt:
		expressionStatement := *assign
		expressionStatement.X = &typeAssertion
		output.Assign = &expressionStatement
	case *ast.AssignStmt:
		assignmentStatement := *assign
		assignmentStatement.Rhs = []ast.Expr{&typeAssertion}
		output.Assign = &assignmentStatement
	}
	output.Body = &ast.BlockStmt{}
	for _, statement := range s.Body.List {
		clause := *statement.(*ast.CaseClause)
		clause.Body = nil
		for _, body := range statement.(*ast.CaseClause).Body {
			clause.Body = append(clause.Body, r.statement(body)...)
		}
		output.Body.List = append(output.Body.List, &clause)
	}
	if len(statements) != 0 {
		output.Init = nil
		return r.withBreakExit(s, &output, []ast.Stmt{&ast.BlockStmt{List: append(statements, &output)}}, exit)
	}
	return r.withBreakExit(s, &output, []ast.Stmt{&output}, exit)
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
