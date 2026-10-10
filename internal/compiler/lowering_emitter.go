package compiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

type loweringEmitter struct {
	unit     *packageUnit
	source   *source
	plan     *functionLoweringPlan
	names    map[string]bool
	values   map[valueID]*ast.Ident
	fmtAlias string
	hoisted  []ast.Stmt
}

func newLoweringEmitter(
	unit *packageUnit,
	source *source,
	plan *functionLoweringPlan,
) *loweringEmitter {
	return &loweringEmitter{
		unit: unit, source: source, plan: plan, names: plan.function.names,
		values: make(map[valueID]*ast.Ident),
	}
}

func (e *loweringEmitter) freshName(preferred string) *ast.Ident {
	return ast.NewIdent(freshIdentifier(preferred, e.names))
}

func (e *loweringEmitter) valueName(id valueID, preferred string) *ast.Ident {
	if name := e.values[id]; name != nil {
		return ast.NewIdent(name.Name)
	}
	name := e.freshName(preferred)
	e.values[id] = name
	return ast.NewIdent(name.Name)
}

// expression emits all planned work into block before it returns the Go value.
func (e *loweringEmitter) expression(
	plan *plannedExpression,
	block *ast.BlockStmt,
) ast.Expr {
	if plan == nil {
		return nil
	}
	if plan.before != nil {
		e.operations(plan.before, block)
	}
	if plan.materialized != 0 {
		return e.valueName(plan.materialized, "operand")
	}
	if plan.kind == planComprehensionExpression && len(plan.results) == 1 {
		name := e.valueName(plan.results[0].id, "result")
		for _, identifier := range plan.resultNames {
			identifier.Name = name.Name
		}
	}
	if plan.work != nil {
		e.operations(plan.work, block)
		if plan.kind == planComprehensionExpression {
			for _, builtin := range plan.builtins {
				if plan.exact != nil && builtin.call == plan.exact.appendCall {
					continue
				}
				builtin.call.Fun = e.unit.generatedUniverse(builtin.name, builtin.position)
			}
			e.applyExactComprehension(plan)
		}
		if len(plan.results) == 1 {
			return e.valueName(plan.results[0].id, "result")
		}
	}
	operands := make([]ast.Expr, 0, len(plan.operands))
	for _, operand := range plan.operands {
		operands = append(operands, e.expression(operand, block))
	}
	switch node := plan.source.(type) {
	case *ast.CallExpr:
		if len(operands) != 0 {
			node.Fun, node.Args = operands[0], operands[1:]
		}
	case *ast.BinaryExpr:
		node.X, node.Y = operands[0], operands[1]
	case *ast.IndexExpr:
		node.X, node.Index = operands[0], operands[1]
	case *ast.IndexListExpr:
		node.X, node.Indices = operands[0], operands[1:]
	case *ast.SelectorExpr:
		node.X = operands[0]
	case *ast.SliceExpr:
		node.X = operands[0]
		index := 1
		if node.Low != nil {
			node.Low = operands[index]
			index++
		}
		if node.High != nil {
			node.High = operands[index]
			index++
		}
		if node.Max != nil {
			node.Max = operands[index]
		}
	case *ast.ParenExpr:
		node.X = operands[0]
	case *ast.StarExpr:
		node.X = operands[0]
	case *ast.UnaryExpr:
		node.X = operands[0]
	case *ast.TypeAssertExpr:
		node.X = operands[0]
	case *ast.CompositeLit:
		node.Elts = operands
	case *ast.KeyValueExpr:
		node.Key, node.Value = operands[0], operands[1]
	}
	return plan.source
}

func (e *loweringEmitter) applyExactComprehension(plan *plannedExpression) {
	if plan.exact == nil {
		return
	}
	source := e.valueName(plan.exact.source.id, "source")
	plan.exact.makeCall.Args[1] = call(
		e.unit.generatedUniverse("len", plan.exact.position), source,
	)
	index, ok := plan.exact.outer.Key.(*ast.Ident)
	if !ok || index.Name == "_" {
		index = e.freshName("index")
		plan.exact.outer.Key = index
	}
	plan.exact.assignment.Lhs[0] = &ast.IndexExpr{
		X: plan.exact.assignment.Lhs[0], Index: index,
	}
	plan.exact.assignment.Rhs[0] = plan.exact.appendCall.Args[1]
}

func (e *loweringEmitter) operations(plan *plannedBlock, output *ast.BlockStmt) {
	for _, operation := range plan.operations {
		e.operation(operation, output)
	}
}

func (e *loweringEmitter) emit() {
	body := &ast.BlockStmt{Lbrace: e.plan.function.body.Lbrace, Rbrace: e.plan.function.body.Rbrace}
	e.operations(e.plan.root, body)
	e.plan.function.body.List = append(e.hoisted, body.List...)
}

func (e *loweringEmitter) operation(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	switch operation.kind {
	case planBlockStatement:
		body := &ast.BlockStmt{}
		e.operations(operation.body, body)
		output.List = append(output.List, body)
	case planIfStatement:
		e.ifStatement(operation, output)
	case planRangeStatement:
		e.rangeStatement(operation, output)
	case planForStatement:
		e.forStatement(operation, output)
	case planSwitchStatement:
		e.switchStatement(operation, output)
	case planTypeSwitchStatement:
		e.typeSwitchStatement(operation, output)
	case planSelectStatement:
		e.selectStatement(operation, output)
	case planLabeledStatement:
		e.labeledStatement(operation, output)
	case planSourceStatement:
		e.sourceStatement(operation, output)
	case planEvaluate:
		expression := operation.expressions[0]
		materialized := expression.materialized
		expression.materialized = 0
		if expression.work != nil {
			e.expressionResults(expression, output)
			expression.materialized = materialized
			break
		}
		value := e.expression(expression, output)
		expression.materialized = materialized
		if len(operation.outputs) != 0 {
			output.List = append(output.List, &ast.AssignStmt{
				Lhs: []ast.Expr{e.valueName(operation.outputs[0], operation.preferredName())},
				Tok: token.DEFINE, Rhs: []ast.Expr{value},
			})
		}
	case planBind:
		call := e.expression(operation.expressions[0], output)
		left := make([]ast.Expr, 0, len(operation.outputs))
		for index, id := range operation.outputs {
			preferred := "result"
			if operation.metadata != nil && index == len(operation.outputs)-1 {
				preferred = "err"
			}
			left = append(left, e.valueName(id, preferred))
		}
		output.List = append(output.List, &ast.AssignStmt{
			Lhs: left, Tok: token.DEFINE, Rhs: []ast.Expr{call},
		})
	case planStore:
		value := e.expression(operation.expressions[0], output)
		output.List = append(output.List, &ast.AssignStmt{
			Lhs: []ast.Expr{e.valueName(operation.inputs[0], "condition")},
			Tok: token.ASSIGN, Rhs: []ast.Expr{value},
		})
	case planBranch:
		condition := ast.Expr(nil)
		if operation.errorValue != 0 {
			condition = &ast.BinaryExpr{
				X: e.valueName(operation.errorValue, "err"), OpPos: operation.metadata.Bang,
				Op: token.NEQ, Y: e.unit.generatedUniverse("nil", operation.metadata.Bang),
			}
		} else {
			condition = e.valueName(operation.inputs[0], "condition")
			if operation.operator == token.LOR {
				condition = &ast.UnaryExpr{Op: token.NOT, X: condition}
			}
		}
		body := &ast.BlockStmt{}
		e.operations(operation.body, body)
		output.List = append(output.List, &ast.IfStmt{Cond: condition, Body: body})
	case planReturn:
		output.List = append(output.List, e.errorReturn(operation))
	}
}

func (operation *plannedOperation) preferredName() string {
	if operation.preferred != "" {
		return operation.preferred
	}
	return "operand"
}

func (e *loweringEmitter) labeledStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.LabeledStmt)
	if e.labeledDeclaration(operation, node, output) {
		return
	}
	body := &ast.BlockStmt{}
	e.operations(operation.body, body)
	if len(body.List) == 0 {
		return
	}
	if len(body.List) == 1 && body.List[0] == node.Stmt {
		output.List = append(output.List, node)
		return
	}
	container := body
	if len(body.List) == 1 {
		if block, ok := body.List[0].(*ast.BlockStmt); ok {
			container = block
		}
	}
	last := len(container.List) - 1
	switch container.List[last].(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
		*ast.TypeSwitchStmt, *ast.SelectStmt:
		label := node.Label
		if operation.labelHasGoto {
			control := e.freshName("control")
			rewriteControlBranches(container.List[last], node.Label.Name, control.Name)
			label = control
		}
		container.List[last] = &ast.LabeledStmt{
			Label: label, Colon: node.Colon, Stmt: container.List[last],
		}
		if operation.labelHasGoto {
			node.Stmt = oneStatement(body.List)
			output.List = append(output.List, node)
		} else {
			output.List = append(output.List, body.List...)
		}
	default:
		node.Stmt = oneStatement(body.List)
		output.List = append(output.List, node)
	}
}

func (e *loweringEmitter) labeledDeclaration(
	operation *plannedOperation,
	node *ast.LabeledStmt,
	output *ast.BlockStmt,
) bool {
	if operation.body == nil || len(operation.body.operations) != 1 {
		return false
	}
	planned := operation.body.operations[0]
	assignment, ok := planned.source.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || planned.binding == nil ||
		len(assignment.Lhs)+1 != len(planned.binding.outputs) {
		return false
	}
	block := &ast.BlockStmt{}
	left := make([]ast.Expr, 0, len(planned.binding.outputs))
	slots := make([]ast.Expr, 0, len(assignment.Lhs))
	for index, target := range assignment.Lhs {
		identifier, ok := target.(*ast.Ident)
		if !ok {
			return false
		}
		value := e.plannedValue(planned.binding.outputs[index])
		zero, ok := e.zeroExpression(value.typ, nil, target.Pos())
		if !ok {
			return false
		}
		slot := e.freshName(identifier.Name)
		e.hoisted = append(e.hoisted, &ast.AssignStmt{
			Lhs: []ast.Expr{slot}, Tok: token.DEFINE, Rhs: []ast.Expr{zero},
		})
		e.values[value.id] = slot
		left = append(left, ast.NewIdent(slot.Name))
		slots = append(slots, ast.NewIdent(slot.Name))
	}
	errorID := planned.binding.outputs[len(planned.binding.outputs)-1]
	errorName := e.valueName(errorID, "err")
	errorType := e.unit.generatedUniverse("error", node.Colon)
	e.hoisted = append(e.hoisted, &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{errorName}, Type: errorType,
		}},
	}})
	left = append(left, ast.NewIdent(errorName.Name))
	call := e.expression(planned.binding.expressions[0], block)
	block.List = append(block.List, &ast.AssignStmt{
		Lhs: left, Tok: token.ASSIGN, Rhs: []ast.Expr{call},
	})
	e.operations(planned.after, block)
	node.Stmt = block
	output.List = append(output.List, node, &ast.AssignStmt{
		Lhs: assignment.Lhs, Tok: token.DEFINE, Rhs: slots,
	})
	return true
}

func (e *loweringEmitter) plannedValue(id valueID) plannedValue {
	for _, value := range e.plan.values {
		if value.id == id {
			return value
		}
	}
	return plannedValue{}
}

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
	target.List = append(target.List, node)
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
	node.X = e.expression(operation.expressions[0], output)
	body := &ast.BlockStmt{Lbrace: node.Body.Lbrace, Rbrace: node.Body.Rbrace}
	e.operations(operation.body, body)
	node.Body = body
	output.List = append(output.List, node)
}

func (e *loweringEmitter) forStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.ForStmt)
	target := output
	if len(operation.headerBindings) != 0 {
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
	} else if operation.init != nil && !blockHasPlannedWork(operation.init) &&
		len(operation.init.operations) == 1 {
		node.Init = operation.init.operations[0].source
	} else if operation.init != nil {
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}
	body := &ast.BlockStmt{Lbrace: node.Body.Lbrace, Rbrace: node.Body.Rbrace}
	postLowered := operation.post != nil && blockHasPlannedWork(operation.post)
	if postLowered {
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
	target.List = append(target.List, node)
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
	if assignment, ok := operation.source.(*ast.AssignStmt); ok &&
		len(operation.places) != 0 && operationHasExpressionWork(operation) {
		left := make([]ast.Expr, 0, len(operation.places))
		for _, place := range operation.places {
			left = append(left, e.preparePlace(place, output))
		}
		assignment.Lhs = left
	}
	if operation.before != nil {
		e.operations(operation.before, output)
	}
	switch node := operation.source.(type) {
	case *ast.ReturnStmt:
		results := make([]ast.Expr, 0, len(operation.expressions))
		for _, expression := range operation.expressions {
			results = append(results, e.expressionResults(expression, output)...)
		}
		if len(operation.failureCommas) != 0 {
			results = e.failureResults(results, operation.failureCommas, output)
			delete(e.source.FailureReturns, node)
		}
		node.Results = results
		output.List = append(output.List, node)
	case *ast.ExprStmt:
		values := e.expressionResults(operation.expressions[0], output)
		if len(values) != 0 {
			node.X = values[0]
			output.List = append(output.List, node)
		}
	case *ast.AssignStmt:
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
		if node.Tok == token.DEFINE && len(operation.expressions) == 1 &&
			e.bindSourceTargets(operation.expressions[0], node.Lhs) {
			e.operations(operation.expressions[0].work, output)
			return
		}
		results := make([]ast.Expr, 0, len(operation.expressions))
		for _, expression := range operation.expressions {
			results = append(results, e.expressionResults(expression, output)...)
		}
		node.Rhs = results
		output.List = append(output.List, node)
	case *ast.DeclStmt:
		if len(operation.declarations) > 1 && declarationsHaveWork(operation.declarations) {
			e.splitDeclaration(node, operation, output)
			return
		}
		if operation.binding != nil {
			declaration := node.Decl.(*ast.GenDecl)
			value := declaration.Specs[0].(*ast.ValueSpec)
			value.Names = append(value.Names, e.valueName(
				operation.binding.outputs[len(operation.binding.outputs)-1], "err",
			))
			value.Values = []ast.Expr{e.expression(operation.binding.expressions[0], output)}
			output.List = append(output.List, node)
			e.operations(operation.after, output)
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
		value := planned.source
		if value.Doc != nil {
			for _, comment := range value.Doc.List {
				comment.Slash = value.Pos() - 1
			}
		}
		declaration := &ast.GenDecl{
			TokPos: value.Pos(), Tok: general.Tok, Specs: []ast.Spec{value},
		}
		if index == 0 {
			declaration.Doc = general.Doc
		}
		before := len(output.List)
		if planned.before != nil {
			e.operations(planned.before, output)
		}
		if planned.binding != nil {
			value.Names = append(value.Names, e.valueName(
				planned.binding.outputs[len(planned.binding.outputs)-1], "err",
			))
			value.Values = []ast.Expr{e.expression(planned.binding.expressions[0], output)}
			output.List = append(output.List, &ast.DeclStmt{Decl: declaration})
			e.operations(planned.after, output)
		} else {
			results := []ast.Expr(nil)
			for _, expression := range planned.expressions {
				results = append(results, e.expressionResults(expression, output)...)
			}
			value.Values = results
			output.List = append(output.List, &ast.DeclStmt{Decl: declaration})
		}
		declarationIndex := len(output.List) - 1
		for item := before; item < len(output.List); item++ {
			if statement, ok := output.List[item].(*ast.DeclStmt); ok && statement.Decl == declaration {
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
}

func (e *loweringEmitter) preparePlace(
	place *plannedPlace,
	output *ast.BlockStmt,
) ast.Expr {
	if place == nil {
		return nil
	}
	operands := make([]ast.Expr, 0, len(place.operands))
	for index, operand := range place.operands {
		if place.address[index] {
			address := &plannedExpression{
				kind:     planUnaryExpression,
				source:   &ast.UnaryExpr{Op: token.AND, X: operand.source},
				typ:      place.values[index].typ,
				operands: []*plannedExpression{operand},
			}
			pointer := e.emitTypedBind(place.values[index], address, output)
			operands = append(operands, &ast.StarExpr{X: pointer})
			continue
		}
		if place.preserve[index] {
			operands = append(operands, e.expression(operand, output))
			continue
		}
		operands = append(operands, e.emitTypedBind(place.values[index], operand, output))
	}
	switch node := place.source.(type) {
	case *ast.ParenExpr:
		if len(operands) != 0 {
			node.X = operands[0]
		}
	case *ast.StarExpr:
		node.X = operands[0]
	case *ast.SelectorExpr:
		node.X = operands[0]
	case *ast.IndexExpr:
		node.X, node.Index = operands[0], operands[1]
	}
	return place.source
}

func (e *loweringEmitter) bindSourceTargets(
	expression *plannedExpression,
	targets []ast.Expr,
) bool {
	if expression == nil || expression.kind != planPropagationExpression ||
		len(expression.results) != len(targets) {
		return false
	}
	for index, target := range targets {
		identifier, ok := target.(*ast.Ident)
		if !ok || identifier.Name == "_" {
			return false
		}
		e.values[expression.results[index].id] = identifier
	}
	return true
}

func (e *loweringEmitter) emitReceiveStore(
	communication *plannedCommunication,
	output *ast.BlockStmt,
) {
	if communication == nil || len(communication.left) == 0 {
		return
	}
	right := make([]ast.Expr, 0, len(communication.receiveValues))
	for _, value := range communication.receiveValues {
		right = append(right, e.valueName(value.id, "received"))
	}
	output.List = append(output.List, &ast.AssignStmt{
		Lhs: communication.left, Tok: communication.token, Rhs: right,
	})
}

func (e *loweringEmitter) emitTypedBind(
	value plannedValue,
	plan *plannedExpression,
	output *ast.BlockStmt,
) ast.Expr {
	expression := e.expression(plan, output)
	if plan != nil && plan.typ != nil {
		if basic, ok := types.Unalias(plan.typ).(*types.Basic); ok &&
			basic.Info()&types.IsUntyped != 0 && !plannedExpressionHasWork(plan) {
			return expression
		}
	}
	if value.typ != nil {
		if basic, ok := types.Unalias(value.typ).(*types.Basic); ok &&
			basic.Info()&types.IsUntyped != 0 {
			return expression
		}
	}
	name := e.valueName(value.id, "operand")
	typeExpression := e.typeExpression(value.typ)
	if typeExpression != nil {
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{name}, Type: typeExpression,
				Values: []ast.Expr{expression},
			}},
		}})
	} else {
		output.List = append(output.List, &ast.AssignStmt{
			Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
		})
	}
	return ast.NewIdent(name.Name)
}

func (e *loweringEmitter) typeExpression(typ types.Type) ast.Expr {
	if typ == nil {
		return nil
	}
	text := types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil || pkg == e.unit.typed {
			return ""
		}
		for _, specification := range e.source.File.Imports {
			path, err := strconv.Unquote(specification.Path.Value)
			if err != nil || path != pkg.Path() {
				continue
			}
			if specification.Name == nil {
				return pkg.Name()
			}
			if specification.Name.Name != "_" && specification.Name.Name != "." {
				return specification.Name.Name
			}
		}
		name := freshASTIdentifier(e.source.File, pkg.Name())
		if name == pkg.Name() {
			astutil.AddImport(e.unit.fs, e.source.File, pkg.Path())
		} else {
			astutil.AddNamedImport(e.unit.fs, e.source.File, name, pkg.Path())
		}
		return name
	})
	expression, err := parser.ParseExpr(text)
	if err != nil {
		return nil
	}
	return expression
}

func (e *loweringEmitter) expressionResults(
	plan *plannedExpression,
	block *ast.BlockStmt,
) []ast.Expr {
	if plan != nil && plan.work != nil && len(plan.results) != 1 {
		e.operations(plan.work, block)
		result := make([]ast.Expr, 0, len(plan.results))
		for _, value := range plan.results {
			result = append(result, e.valueName(value.id, "result"))
		}
		return result
	}
	return []ast.Expr{e.expression(plan, block)}
}

func (e *loweringEmitter) errorReturn(operation *plannedOperation) ast.Stmt {
	count := e.plan.function.resultType.Len() - 1
	results := make([]ast.Expr, 0, count+1)
	for index := 0; index < count; index++ {
		typ := e.plan.function.resultType.At(index).Type()
		resultType := e.plan.function.resultAST[index]
		if value, ok := e.zeroExpression(typ, resultType, operation.metadata.Bang); ok {
			results = append(results, value)
			continue
		}
		name := e.freshName("zero")
		specification := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: resultType}
		e.unit.generatedValues[specification] = true
		results = append(results, name)
	}
	returnedError := ast.Expr(e.valueName(operation.errorValue, "err"))
	if !operation.metadata.Transparent {
		returnedError = call(
			e.unit.generatedObject(e.formatQualifier(), "fmt", "Errorf", operation.metadata.Bang),
			&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(operation.metadata.Name + ": %w")},
			returnedError,
		)
	}
	results = append(results, returnedError)
	return &ast.ReturnStmt{Results: results}
}

func (e *loweringEmitter) zeroExpression(
	value types.Type,
	typeExpression ast.Expr,
	position token.Pos,
) (ast.Expr, bool) {
	underlying := types.Unalias(value)
	if named, ok := underlying.(*types.Named); ok {
		underlying = named.Underlying()
	}
	switch item := underlying.(type) {
	case *types.Basic:
		switch {
		case item.Info()&types.IsBoolean != 0:
			return e.unit.generatedUniverse("false", position), true
		case item.Info()&types.IsString != 0:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}, true
		case item.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}, true
		case item.Kind() == types.UnsafePointer || item.Kind() == types.UntypedNil:
			return e.unit.generatedUniverse("nil", position), true
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan,
		*types.Signature, *types.Interface:
		return e.unit.generatedUniverse("nil", position), true
	case *types.Array, *types.Struct:
		return &ast.CompositeLit{Type: typeExpression}, true
	}
	return nil, false
}

func (e *loweringEmitter) formatQualifier() string {
	if e.fmtAlias != "" {
		return e.fmtAlias
	}
	for _, specification := range e.source.File.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "fmt" {
			continue
		}
		if specification.Name == nil {
			e.fmtAlias = "fmt"
			return e.fmtAlias
		}
		if specification.Name.Name == "_" {
			e.fmtAlias = freshASTIdentifier(e.source.File, "fmt")
			if e.fmtAlias == "fmt" {
				specification.Name = nil
			} else {
				specification.Name = ast.NewIdent(e.fmtAlias)
			}
			return e.fmtAlias
		}
		if specification.Name.Name == "." {
			return ""
		}
		e.fmtAlias = specification.Name.Name
		return e.fmtAlias
	}
	e.fmtAlias = freshASTIdentifier(e.source.File, "fmt")
	if e.fmtAlias == "fmt" {
		astutil.AddImport(e.unit.fs, e.source.File, "fmt")
	} else {
		astutil.AddNamedImport(e.unit.fs, e.source.File, e.fmtAlias, "fmt")
	}
	return e.fmtAlias
}
