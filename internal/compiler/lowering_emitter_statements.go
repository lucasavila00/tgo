package compiler

import (
	"go/ast"
	"go/token"
)

func (e *loweringEmitter) typeSwitchStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.TypeSwitchStmt)
	target := output
	if operation.init != nil {
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}
	if operation.test != nil && len(operation.test.operations) == 1 {
		planned := operation.test.operations[0]
		switch source := planned.source.(type) {
		case *ast.AssignStmt:
			if len(planned.expressions) != 1 {
				break
			}
			assignment := source
			assignment.Rhs = []ast.Expr{e.expression(planned.expressions[0], target)}
			node.Assign = assignment
		case *ast.ExprStmt:
			if len(planned.expressions) == 1 {
				source.X = e.expression(planned.expressions[0], target)
				node.Assign = source
			}
		}
	}
	for index, item := range node.Body.List {
		clause := item.(*ast.CaseClause)
		body := &ast.BlockStmt{}
		e.operations(operation.cases[index], body)
		clause.Body = body.List
	}
	e.appendControl(target, operation.target, node)
}

func (e *loweringEmitter) ifStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.IfStmt)
	target := output
	if operation.init != nil {
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}
	node.Cond = e.expression(operation.expressions[0], target)
	body := &ast.BlockStmt{Lbrace: node.Body.Lbrace, Rbrace: node.Body.Rbrace}
	e.operations(operation.body, body)
	node.Body = body
	if operation.otherwise != nil {
		alternative := &ast.BlockStmt{}
		e.operations(operation.otherwise, alternative)
		if len(alternative.List) == 1 {
			node.Else = alternative.List[0]
		} else {
			node.Else = alternative
		}
	}
	target.List = append(target.List, node)
}

func (e *loweringEmitter) rangeStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.RangeStmt)
	target := output
	if plannedExpressionHasWork(operation.expressions[0]) {
		target = &ast.BlockStmt{}
		output.List = append(output.List, target)
	}
	node.X = e.expression(operation.expressions[0], target)
	body := &ast.BlockStmt{Lbrace: node.Body.Lbrace, Rbrace: node.Body.Rbrace}
	e.operations(operation.body, body)
	node.Body = body
	e.appendControl(target, operation.target, node)
}

func (e *loweringEmitter) forStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.ForStmt)
	target := output
	switch {
	case len(operation.headerBindings) != 0:
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		left := make([]ast.Expr, 0, len(operation.headerBindings))
		right := make([]ast.Expr, 0, len(operation.headerBindings))
		for _, binding := range operation.headerBindings {
			left = append(left, binding.target)
			right = append(right, e.valueName(binding.value.id, "initializer"))
		}
		node.Init = &ast.AssignStmt{Lhs: left, Tok: token.DEFINE, Rhs: right}
		for _, statement := range wrapper.List {
			positionGeneratedStatement(statement, node.For)
		}
		output.List = append(output.List, wrapper)
		target = wrapper
	case operation.init != nil && !blockHasPlannedWork(operation.init) &&
		len(operation.init.operations) == 1:
		node.Init = operation.init.operations[0].source
	case operation.init != nil:
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}
	body := &ast.BlockStmt{Lbrace: node.Body.Lbrace, Rbrace: node.Body.Rbrace}
	postLowered := operation.post != nil && blockHasPlannedWork(operation.post)
	if postLowered {
		if target == output {
			target = &ast.BlockStmt{}
			output.List = append(output.List, target)
		}
		pending := e.freshName("post")
		target.List = append(target.List, &ast.AssignStmt{
			Lhs: []ast.Expr{pending}, Tok: token.DEFINE,
			Rhs: []ast.Expr{e.unit.generatedUniverse("false", node.For)},
		})
		postBody := &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{e.unit.generatedUniverse("false", node.For)},
		}}}
		e.operations(operation.post, postBody)
		body.List = append(body.List, &ast.IfStmt{
			Cond: ast.NewIdent(pending.Name), Body: postBody,
		})
		node.Post = &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{e.unit.generatedUniverse("true", node.For)},
		}
	}
	if len(operation.expressions) != 0 &&
		(postLowered || plannedExpressionHasWork(operation.expressions[0])) {
		condition := e.expression(operation.expressions[0], body)
		body.List = append(body.List, &ast.IfStmt{
			Cond: &ast.UnaryExpr{Op: token.NOT, X: condition},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}},
		})
		node.Cond = nil
	}
	e.operations(operation.body, body)
	node.Body = body
	e.appendControl(target, operation.target, node)
}

func (e *loweringEmitter) appendControl(
	output *ast.BlockStmt,
	target targetID,
	statement ast.Stmt,
) {
	if label := e.targets[target]; label != nil {
		output.List = append(output.List, &ast.LabeledStmt{
			Label: label, Colon: e.plan.targets[target].position, Stmt: statement,
		})
		return
	}
	output.List = append(output.List, statement)
}

func blockHasPlannedWork(block *plannedBlock) bool {
	if block == nil {
		return false
	}
	for _, operation := range block.operations {
		for _, expression := range operation.expressions {
			if plannedExpressionHasWork(expression) {
				return true
			}
		}
	}
	return false
}

func plannedExpressionHasWork(expression *plannedExpression) bool {
	if expression == nil {
		return false
	}
	if expression.work != nil && len(expression.work.operations) != 0 ||
		expression.before != nil && len(expression.before.operations) != 0 {
		return true
	}
	for _, operand := range expression.operands {
		if plannedExpressionHasWork(operand) {
			return true
		}
	}
	return false
}

func (e *loweringEmitter) sourceStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	if e.emitPlannedAssignment(operation, output) {
		return
	}
	if operation.before != nil {
		e.operations(operation.before, output)
	}
	switch node := operation.source.(type) {
	case *ast.ReturnStmt:
		e.emitSourceReturn(node, operation, output)
	case *ast.ExprStmt:
		values := e.expressionResults(operation.expressions[0], output)
		if len(values) != 0 {
			node.X = values[0]
			output.List = append(output.List, node)
		}
	case *ast.IncDecStmt:
		output.List = append(output.List, node)
	case *ast.AssignStmt:
		e.emitSourceAssignment(node, operation, output)
	case *ast.DeclStmt:
		e.emitSourceDeclaration(node, operation, output)
	case *ast.SendStmt:
		node.Chan = e.expression(operation.expressions[0], output)
		node.Value = e.expression(operation.expressions[1], output)
		output.List = append(output.List, node)
	case *ast.GoStmt:
		node.Call = e.expression(operation.expressions[0], output).(*ast.CallExpr)
		output.List = append(output.List, node)
	case *ast.DeferStmt:
		node.Call = e.expression(operation.expressions[0], output).(*ast.CallExpr)
		output.List = append(output.List, node)
	default:
		if operation.source != nil {
			output.List = append(output.List, operation.source)
		}
	}
}

func (e *loweringEmitter) emitSourceReturn(
	node *ast.ReturnStmt,
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	results := []ast.Expr(nil)
	for _, expression := range operation.expressions {
		results = append(results, e.expressionResults(expression, output)...)
	}
	if len(operation.failureCommas) != 0 {
		results = e.failureResults(results, operation.failureCommas, output)
		delete(e.source.FailureReturns, node)
	}
	node.Results = results
	output.List = append(output.List, node)
}

func (e *loweringEmitter) emitSourceAssignment(
	node *ast.AssignStmt,
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	if operation.binding != nil {
		call := e.expression(operation.binding.expressions[0], output)
		left := append([]ast.Expr(nil), node.Lhs...)
		left = append(left, e.valueName(
			operation.binding.outputs[len(operation.binding.outputs)-1], "err",
		))
		node.Lhs, node.Rhs = left, []ast.Expr{call}
		output.List = append(output.List, node)
		e.operations(operation.after, output)
		return
	}
	results := make([]ast.Expr, 0, len(operation.expressions))
	for _, expression := range operation.expressions {
		results = append(results, e.expressionResults(expression, output)...)
	}
	node.Rhs = results
	output.List = append(output.List, node)
}

func (e *loweringEmitter) emitSourceDeclaration(
	node *ast.DeclStmt,
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	e.recordTypeDefinitions(node, output)
	if declarationsHaveWork(operation.declarations) {
		e.splitDeclaration(node, operation, output)
		return
	}
	if operation.binding != nil {
		e.emitBoundDeclaration(node, operation, output)
		return
	}
	expressionIndex := 0
	if declaration, ok := node.Decl.(*ast.GenDecl); ok {
		for _, specification := range declaration.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			results := []ast.Expr(nil)
			for range value.Values {
				results = append(results, e.expressionResults(
					operation.expressions[expressionIndex], output,
				)...)
				expressionIndex++
			}
			value.Values = results
		}
	}
	output.List = append(output.List, node)
}

func (e *loweringEmitter) emitBoundDeclaration(
	node *ast.DeclStmt,
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	declaration := node.Decl.(*ast.GenDecl)
	value := declaration.Specs[0].(*ast.ValueSpec)
	value.Names = append(value.Names, e.valueName(
		operation.binding.outputs[len(operation.binding.outputs)-1], "err",
	))
	value.Values = []ast.Expr{e.expression(operation.binding.expressions[0], output)}
	output.List = append(output.List, node)
	e.operations(operation.after, output)
}

func (e *loweringEmitter) recordTypeDefinitions(
	statement *ast.DeclStmt,
	output *ast.BlockStmt,
) {
	declaration, ok := statement.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.TYPE {
		return
	}
	for _, specification := range declaration.Specs {
		typeSpecification, ok := specification.(*ast.TypeSpec)
		if !ok {
			continue
		}
		e.typeDefinitionSites[typeSpecification.Name] = emittedTypeDefinition{
			block: output, index: len(output.List),
		}
	}
}

func operationHasExpressionWork(operation *plannedOperation) bool {
	if operation.before != nil {
		return true
	}
	for _, expression := range operation.expressions {
		if plannedExpressionHasWork(expression) {
			return true
		}
	}
	return false
}

func (e *loweringEmitter) failureResults(
	results []ast.Expr,
	commas []token.Pos,
	output *ast.BlockStmt,
) []ast.Expr {
	if len(results) != 1 {
		e.unit.failAt(commas[0], "failure return error expression must produce one value")
		return results
	}
	if len(commas) >= e.plan.function.resultType.Len() {
		index := e.plan.function.resultType.Len() - 1
		if index < 0 {
			index = 0
		}
		e.unit.failAt(commas[index], "failure return has more commas than preceding results")
		return results
	}
	last := e.plan.function.resultType.Len() - 1
	if last < 0 || !isPredeclaredError(e.plan.function.resultType.At(last).Type()) {
		e.unit.failAt(commas[0], "failure return function must end in the Go error type")
		return results
	}
	zeros := make([]ast.Expr, 0, len(commas)+1)
	for index := range len(commas) {
		typ := e.plan.function.resultType.At(index).Type()
		typeExpression := e.plan.function.resultAST[index]
		if value, ok := e.zeroExpression(typ, typeExpression, commas[0]); ok {
			zeros = append(zeros, value)
			continue
		}
		name := e.freshName("zero")
		specification := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: typeExpression}
		e.unit.generatedValues[specification] = true
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{specification},
		}})
		zeros = append(zeros, name)
	}
	return append(zeros, results[0])
}

func declarationsHaveWork(declarations []*plannedDeclaration) bool {
	for _, declaration := range declarations {
		for _, expression := range declaration.expressions {
			if plannedExpressionHasWork(expression) {
				return true
			}
		}
	}
	return false
}

func (e *loweringEmitter) splitDeclaration(
	node *ast.DeclStmt,
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	general := node.Decl.(*ast.GenDecl)
	for index, planned := range operation.declarations {
		e.emitSplitDeclaration(general, planned, index == 0, output)
	}
}

func (e *loweringEmitter) emitSplitDeclaration(
	general *ast.GenDecl,
	planned *plannedDeclaration,
	first bool,
	output *ast.BlockStmt,
) {
	value := planned.source
	positionDeclarationComments(value)
	declaration := &ast.GenDecl{
		TokPos: value.Pos(), Tok: general.Tok, Specs: []ast.Spec{value},
	}
	if first {
		declaration.Doc = general.Doc
	}
	before := len(output.List)
	if planned.before != nil {
		e.operations(planned.before, output)
	}
	e.emitSplitDeclarationValue(value, declaration, planned, output)
	e.positionSplitDeclaration(value, declaration, before, output)
}

func positionDeclarationComments(value *ast.ValueSpec) {
	if value.Doc == nil {
		return
	}
	for _, comment := range value.Doc.List {
		comment.Slash = value.Pos() - 1
	}
}

func (e *loweringEmitter) emitSplitDeclarationValue(
	value *ast.ValueSpec,
	declaration *ast.GenDecl,
	planned *plannedDeclaration,
	output *ast.BlockStmt,
) {
	if planned.binding != nil {
		value.Names = append(value.Names, e.valueName(
			planned.binding.outputs[len(planned.binding.outputs)-1], "err",
		))
		value.Values = []ast.Expr{e.expression(planned.binding.expressions[0], output)}
		output.List = append(output.List, &ast.DeclStmt{Decl: declaration})
		e.operations(planned.after, output)
		return
	}
	results := []ast.Expr(nil)
	for _, expression := range planned.expressions {
		results = append(results, e.expressionResults(expression, output)...)
	}
	value.Values = results
	output.List = append(output.List, &ast.DeclStmt{Decl: declaration})
}

func (e *loweringEmitter) positionSplitDeclaration(
	value *ast.ValueSpec,
	declaration *ast.GenDecl,
	before int,
	output *ast.BlockStmt,
) {
	declarationIndex := len(output.List) - 1
	for item := before; item < len(output.List); item++ {
		statement, ok := output.List[item].(*ast.DeclStmt)
		if ok && statement.Decl == declaration {
			declarationIndex = item
			break
		}
	}
	for _, statement := range output.List[before:declarationIndex] {
		positionGeneratedStatement(statement, value.Pos())
	}
	positionGeneratedStatement(output.List[declarationIndex], value.Pos())
	position := value.End()
	if value.Comment != nil {
		position = value.Comment.End() + 1
	}
	for _, statement := range output.List[declarationIndex+1:] {
		positionGeneratedStatement(statement, position)
	}
}
