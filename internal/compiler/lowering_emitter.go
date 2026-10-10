package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

type loweringEmitter struct {
	unit                *packageUnit
	source              *source
	plan                *functionLoweringPlan
	names               map[string]bool
	values              map[valueID]*ast.Ident
	places              map[placeID]ast.Expr
	targets             map[targetID]*ast.Ident
	typeAliases         map[*ast.BlockStmt]map[types.Type]*ast.Ident
	renamedTypeBlockers map[types.Object]string
	typeDefinitionSites map[*ast.Ident]emittedTypeDefinition
	typeObjectAliases   map[types.Object]*ast.Ident
	forcedQualifiers    map[*types.Package]string
	fmtAlias            string
}

type emittedTypeDefinition struct {
	block *ast.BlockStmt
	index int
}

func newLoweringEmitter(
	unit *packageUnit,
	source *source,
	plan *functionLoweringPlan,
) *loweringEmitter {
	if source.LoweringTypeAliases == nil {
		source.LoweringTypeAliases = make(map[types.Object]*ast.Ident)
	}
	emitter := &loweringEmitter{
		unit: unit, source: source, plan: plan, names: plan.function.names,
		values: make(map[valueID]*ast.Ident), targets: make(map[targetID]*ast.Ident),
		places:              make(map[placeID]ast.Expr),
		typeAliases:         make(map[*ast.BlockStmt]map[types.Type]*ast.Ident),
		renamedTypeBlockers: make(map[types.Object]string),
		typeDefinitionSites: make(map[*ast.Ident]emittedTypeDefinition),
		typeObjectAliases:   source.LoweringTypeAliases,
		forcedQualifiers:    make(map[*types.Package]string),
	}
	for _, alias := range source.LoweringTypeAliases {
		emitter.names[alias.Name] = true
	}
	for id, target := range plan.targets {
		if target.label != "" {
			emitter.targets[id] = ast.NewIdent(target.label)
		}
	}
	return emitter
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
	if plan.place != nil {
		return e.places[plan.place.id]
	}
	if plan.materialized != 0 {
		value := ast.Expr(e.valueName(plan.materialized, "operand"))
		return e.booleanContextAdapter(plan, value)
	}
	if plan.before != nil {
		e.operations(plan.before, block)
	}
	if plan.kind == planComprehensionExpression && len(plan.results) == 1 {
		name := e.valueName(plan.results[0].id, "result")
		for _, identifier := range plan.resultNames {
			identifier.Name = name.Name
		}
	}
	if plan.work != nil {
		e.emitExpressionWork(plan, block)
		if len(plan.results) == 1 {
			value := ast.Expr(e.valueName(plan.results[0].id, "result"))
			return e.booleanContextAdapter(plan, value)
		}
	}
	operands := make([]ast.Expr, 0, len(plan.operands))
	for _, operand := range plan.operands {
		operands = append(operands, e.expression(operand, block))
	}
	e.applyExpressionOperands(plan.source, operands)
	return e.booleanContextAdapter(plan, plan.source)
}

func (e *loweringEmitter) booleanContextAdapter(
	plan *plannedExpression,
	value ast.Expr,
) ast.Expr {
	if !plan.booleanAdapter {
		return value
	}
	return &ast.BinaryExpr{
		X: value, Op: token.EQL,
		Y: &ast.BinaryExpr{
			X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: token.EQL,
			Y: &ast.BasicLit{Kind: token.INT, Value: "0"},
		},
	}
}

func (e *loweringEmitter) emitExpressionWork(
	plan *plannedExpression,
	block *ast.BlockStmt,
) {
	e.operations(plan.work, block)
	if plan.kind != planComprehensionExpression {
		return
	}
	for _, builtin := range plan.builtins {
		if plan.exact != nil && builtin.call == plan.exact.appendCall {
			continue
		}
		builtin.call.Fun = e.unit.generatedUniverse(builtin.name, builtin.position)
	}
	e.applyExactComprehension(plan)
}

func (e *loweringEmitter) applyExpressionOperands(source ast.Expr, operands []ast.Expr) {
	switch node := source.(type) {
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
		applySliceOperands(node, operands)
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
}

func applySliceOperands(node *ast.SliceExpr, operands []ast.Expr) {
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
}

func (e *loweringEmitter) applyExactComprehension(
	plan *plannedExpression,
) {
	if plan.exact == nil {
		return
	}
	source := e.valueName(plan.exact.source.id, "source")
	plan.exact.makeCall.Args[1] = call(
		e.unit.generatedUniverse("len", plan.exact.position), source,
	)
	if plan.exact.identity {
		return
	}
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
	e.plan.function.body.List = body.List
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
	default:
		e.valueOperation(operation, output)
	}
}

func (e *loweringEmitter) valueOperation(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	switch operation.kind {
	case planJump:
		e.emitJump(operation, output)
	case planEvaluate:
		e.emitEvaluation(operation, output)
	case planBind:
		e.emitBinding(operation, output)
	case planStore:
		value := e.expression(operation.expressions[0], output)
		output.List = append(output.List, &ast.AssignStmt{
			Lhs: []ast.Expr{e.valueName(operation.inputs[0], "condition")},
			Tok: token.ASSIGN, Rhs: []ast.Expr{value},
		})
	case planBranch:
		var condition ast.Expr
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
		output.List = append(output.List, e.errorReturn(operation, output))
	case planCopy:
		output.List = append(output.List, &ast.ExprStmt{X: call(
			e.unit.generatedUniverse("copy", operation.source.Pos()),
			operation.copyTarget, e.valueName(operation.inputs[0], "source"),
		)})
	case planPreparePlace:
		place := operation.places[0]
		e.places[place.id] = e.preparePlace(place, output)
	}
}

func (e *loweringEmitter) emitJump(operation *plannedOperation, output *ast.BlockStmt) {
	branch := operation.source.(*ast.BranchStmt)
	var label *ast.Ident
	if branch.Label != nil && operation.target != 0 {
		label = e.targets[operation.target]
	}
	output.List = append(output.List, &ast.BranchStmt{
		TokPos: branch.TokPos, Tok: branch.Tok, Label: label,
	})
}

func (e *loweringEmitter) emitEvaluation(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	expression := operation.expressions[0]
	materialized := expression.materialized
	expression.materialized = 0
	if expression.work != nil {
		e.expressionResults(expression, output)
		expression.materialized = materialized
		return
	}
	value := e.expression(expression, output)
	expression.materialized = materialized
	if len(operation.outputs) == 0 {
		return
	}
	planned := e.plannedValue(operation.outputs[0])
	if operation.preferred != "" && e.values[planned.id] == nil {
		e.values[planned.id] = e.freshName(operation.preferred)
	}
	planned.explicit = expression.contextual
	e.emitTypedExpressionBind(planned, value, planned.explicit, output)
}

func (e *loweringEmitter) emitBinding(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	call := e.expression(operation.expressions[0], output)
	if len(operation.outputs) == 1 {
		planned := e.plannedValue(operation.outputs[0])
		expression := operation.expressions[0]
		planned.explicit = expression.contextual
		e.emitTypedExpressionBind(planned, call, planned.explicit, output)
		return
	}
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
}

func (e *loweringEmitter) labeledStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.LabeledStmt)
	if operation.body != nil && len(operation.body.operations) == 1 {
		child := operation.body.operations[0]
		if child.source == node.Stmt && !operationHasPlannedWork(child) {
			output.List = append(output.List, node)
			return
		}
	}
	separateEntry := operation.labelHasGoto && e.controlNeedsWrapper(operation)
	e.configureLabeledTarget(operation, node, separateEntry)
	body := &ast.BlockStmt{}
	e.operations(operation.body, body)
	if len(body.List) == 0 {
		return
	}
	if operation.controlTarget == 0 && operation.labelHasGoto {
		node.Stmt = &ast.EmptyStmt{Implicit: true}
		output.List = append(output.List, node)
		output.List = append(output.List, body.List...)
		return
	}
	if operation.controlTarget == 0 || separateEntry {
		node.Stmt = oneStatement(body.List)
		output.List = append(output.List, node)
		return
	}
	output.List = append(output.List, body.List...)
}

func (e *loweringEmitter) configureLabeledTarget(
	operation *plannedOperation,
	node *ast.LabeledStmt,
	separateEntry bool,
) {
	if operation.controlTarget == 0 {
		return
	}
	if !separateEntry {
		e.targets[operation.controlTarget] = ast.NewIdent(node.Label.Name)
		return
	}
	if blockUsesTarget(operation.body, operation.controlTarget) {
		e.targets[operation.controlTarget] = e.freshName("control")
		return
	}
	delete(e.targets, operation.controlTarget)
}

func blockUsesTarget(block *plannedBlock, target targetID) bool {
	if block == nil {
		return false
	}
	for _, operation := range block.operations {
		if operation.kind == planJump && operation.target == target {
			return true
		}
		for _, child := range []*plannedBlock{
			operation.init, operation.test, operation.body, operation.post,
			operation.otherwise, operation.before, operation.after,
		} {
			if blockUsesTarget(child, target) {
				return true
			}
		}
		for _, child := range operation.cases {
			if blockUsesTarget(child, target) {
				return true
			}
		}
	}
	return false
}

func operationHasPlannedWork(operation *plannedOperation) bool {
	if operation == nil || operationHasExpressionWork(operation) {
		return operation != nil
	}
	for _, block := range []*plannedBlock{
		operation.init, operation.test, operation.body, operation.post,
		operation.otherwise, operation.before, operation.after,
	} {
		if blockHasPlannedWork(block) {
			return true
		}
	}
	for _, block := range operation.cases {
		if blockHasPlannedWork(block) {
			return true
		}
	}
	return false
}

func (e *loweringEmitter) controlNeedsWrapper(operation *plannedOperation) bool {
	if operation.body == nil || len(operation.body.operations) != 1 {
		return false
	}
	control := operation.body.operations[0]
	switch control.kind {
	case planForStatement, planRangeStatement, planSwitchStatement,
		planTypeSwitchStatement, planSelectStatement:
		return operationHasPlannedWork(control)
	}
	return false
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
	e.prepareSourceStatementPlaces(operation, output)
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

func (e *loweringEmitter) prepareSourceStatementPlaces(
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
	if update, ok := operation.source.(*ast.IncDecStmt); ok &&
		len(operation.places) == 1 && operationHasExpressionWork(operation) {
		update.X = e.preparePlace(operation.places[0], output)
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

func (e *loweringEmitter) preparePlace(
	place *plannedPlace,
	output *ast.BlockStmt,
) ast.Expr {
	if place == nil {
		return nil
	}
	switch place.kind {
	case planObjectPlace:
		return place.source
	case planDerefPlace:
		var pointer ast.Expr
		if place.base != nil {
			pointer = e.emitTypedExpressionBind(
				place.values[0], e.preparePlace(place.base, output), false, output,
			)
		} else {
			pointer = e.emitTypedBind(place.values[0], place.container, output)
		}
		return &ast.StarExpr{X: pointer}
	case planFieldPlace:
		selector := place.source.(*ast.SelectorExpr)
		return &ast.SelectorExpr{X: e.preparePlace(place.base, output), Sel: selector.Sel}
	case planArrayIndexPlace:
		base := e.preparePlace(place.base, output)
		index := e.emitTypedBind(place.values[0], place.index, output)
		return &ast.IndexExpr{X: base, Index: index}
	case planSliceIndexPlace, planMapIndexPlace:
		container := e.emitTypedBind(place.values[0], place.container, output)
		index := e.emitTypedBind(place.values[1], place.index, output)
		return &ast.IndexExpr{X: container, Index: index}
	default:
		return place.source
	}
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
	left := communication.left
	if communication.token != token.DEFINE && len(communication.targets) != 0 {
		left = make([]ast.Expr, 0, len(communication.targets))
		for _, place := range communication.targets {
			left = append(left, e.preparePlace(place, output))
		}
	}
	output.List = append(output.List, &ast.AssignStmt{
		Lhs: left, Tok: communication.token, Rhs: right,
	})
}

func (e *loweringEmitter) emitTypedBind(
	value plannedValue,
	plan *plannedExpression,
	output *ast.BlockStmt,
) ast.Expr {
	expression := e.expression(plan, output)
	if retainTypedBindExpression(value, plan) {
		return expression
	}
	explicit := value.explicit || plan != nil && plan.typ != nil && value.typ != nil &&
		!types.Identical(plan.typ, value.typ)
	return e.emitTypedExpressionBind(value, expression, explicit, output)
}

func retainTypedBindExpression(value plannedValue, plan *plannedExpression) bool {
	if plan != nil && plan.retainContext && !plannedExpressionHasWork(plan) {
		return true
	}
	if plan != nil && isUntypedType(plan.typ) && !plannedExpressionHasWork(plan) {
		return true
	}
	return isUntypedType(value.typ)
}

func (e *loweringEmitter) emitTypedExpressionBind(
	value plannedValue,
	expression ast.Expr,
	explicit bool,
	output *ast.BlockStmt,
) ast.Expr {
	name := e.valueName(value.id, "operand")
	var typeExpression ast.Expr
	if explicit {
		typeExpression = e.contextTypeExpression(value, output)
	}
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

func (e *loweringEmitter) contextTypeExpression(
	value plannedValue,
	output *ast.BlockStmt,
) ast.Expr {
	reference := value.typeReference
	if len(reference.blockers) != 0 {
		e.renameTypeReferenceBlockers(value)
		e.emitTypeAliasesBeforeBlockers(value)
		return e.typeExpression(value.typ, value.position)
	}
	if reference.direct {
		return e.typeExpression(value.typ, value.position)
	}
	typ := value.typ
	aliases := e.typeAliases[output]
	if aliases == nil {
		aliases = make(map[types.Type]*ast.Ident)
		e.typeAliases[output] = aliases
	}
	if alias := aliases[typ]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	alias := e.freshName("operandType")
	aliases[typ] = alias
	aliasPosition := output.Lbrace
	if aliasPosition == token.NoPos {
		aliasPosition = e.plan.function.body.Lbrace
	}
	typeExpression := e.typeExpression(typ, aliasPosition)
	declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{
			Name: alias, Assign: aliasPosition, Type: typeExpression,
		}},
	}}
	output.List = append([]ast.Stmt{declaration}, output.List...)
	return ast.NewIdent(alias.Name)
}

func (e *loweringEmitter) emitTypeAliasesBeforeBlockers(value plannedValue) {
	for _, planned := range value.typeReference.blockers {
		if !planned.typeName || e.typeObjectAliases[planned.intended] != nil {
			continue
		}
		if planned.alias.foreignQualified {
			e.ensureQualifiedTypeOwner(planned.intended.Pkg())
			continue
		}
		if planned.alias.packageScope {
			e.emitPackageTypeAlias(planned)
			continue
		}
		site, ok := e.typeDefinitionSites[planned.definition]
		if !ok {
			continue
		}
		alias := e.freshName(planned.intended.Name() + "Type")
		position := planned.alias.before.Pos()
		declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.TYPE, Specs: []ast.Spec{
				e.typeAliasSpecification(alias, planned.alias, position),
			},
		}}
		site.block.List = append(site.block.List, nil)
		copy(site.block.List[site.index+1:], site.block.List[site.index:])
		site.block.List[site.index] = declaration
		for definition, other := range e.typeDefinitionSites {
			if other.block == site.block && other.index >= site.index {
				other.index++
				e.typeDefinitionSites[definition] = other
			}
		}
		e.typeObjectAliases[planned.intended] = alias
	}
}

func (e *loweringEmitter) ensureQualifiedTypeOwner(owner *types.Package) {
	if e.forcedQualifiers[owner] != "" {
		return
	}
	alias := freshASTIdentifier(e.source.File, owner.Name())
	astutil.AddNamedImport(e.unit.fs, e.source.File, alias, owner.Path())
	e.forcedQualifiers[owner] = alias
	e.names[alias] = true
}

func (e *loweringEmitter) emitPackageTypeAlias(planned plannedTypeBlocker) {
	alias := e.freshName(planned.intended.Name() + "Type")
	position := e.source.File.Package
	specification := e.typeAliasSpecification(alias, planned.alias, position)
	declaration := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{specification}}
	index := 0
	for index < len(e.source.File.Decls) {
		imports, ok := e.source.File.Decls[index].(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			break
		}
		index++
	}
	e.source.File.Decls = append(e.source.File.Decls, nil)
	copy(e.source.File.Decls[index+1:], e.source.File.Decls[index:])
	e.source.File.Decls[index] = declaration
	e.typeObjectAliases[planned.intended] = alias
}

func (e *loweringEmitter) typeAliasSpecification(
	alias *ast.Ident,
	action *plannedTypeAliasAction,
	position token.Pos,
) *ast.TypeSpec {
	specification := &ast.TypeSpec{
		Name: alias, Assign: position, Type: e.typeObjectExpression(action.object, position),
	}
	if len(action.parameters) == 0 {
		return specification
	}
	parameters := make([]*ast.Field, 0, len(action.parameters))
	arguments := make([]ast.Expr, 0, len(action.parameters))
	for _, parameter := range action.parameters {
		name := ast.NewIdent(parameter.name)
		parameters = append(parameters, &ast.Field{
			Names: []*ast.Ident{name}, Type: e.typeExpression(parameter.constraint, position),
		})
		arguments = append(arguments, ast.NewIdent(name.Name))
	}
	specification.TypeParams = &ast.FieldList{List: parameters}
	specification.Type = &ast.IndexListExpr{X: specification.Type, Indices: arguments}
	return specification
}

func (e *loweringEmitter) typeObjectExpression(object types.Object, position token.Pos) ast.Expr {
	if object.Pkg() == nil {
		return e.unit.generatedUniverse(object.Name(), position)
	}
	if object.Pkg() == e.unit.typed && object.Parent() != e.unit.typed.Scope() {
		return ast.NewIdent(object.Name())
	}
	qualifier := e.typeOwnerQualifier(object.Pkg())
	return e.unit.generatedObject(
		qualifier, packagePath(object.Pkg()), object.Name(), position,
	)
}

func (e *loweringEmitter) typeOwnerQualifier(owner *types.Package) string {
	if qualifier := e.forcedQualifiers[owner]; qualifier != "" {
		return qualifier
	}
	return e.unit.ownerQualifier(e.source.File, owner)
}

func (e *loweringEmitter) renameTypeReferenceBlockers(value plannedValue) {
	for _, planned := range value.typeReference.blockers {
		if planned.typeName {
			continue
		}
		blocker := planned.object
		name := e.renamedTypeBlockers[blocker]
		if name == "" {
			name = e.freshName(blocker.Name() + "Value").Name
			e.renamedTypeBlockers[blocker] = name
		}
		for _, identifier := range planned.identifiers {
			identifier.Name = name
		}
	}
}

func (e *loweringEmitter) typeExpression(typ types.Type, position token.Pos) ast.Expr {
	if typ == nil {
		return nil
	}
	if _, alias := typ.(*types.Alias); alias {
		typ = types.Unalias(typ)
	}
	switch item := typ.(type) {
	case *types.Basic:
		return e.basicTypeExpression(item, position)
	case *types.Named:
		return e.namedTypeExpression(item, position)
	case *types.Pointer:
		return e.containerTypeExpression(item, position)
	case *types.Slice:
		return e.containerTypeExpression(item, position)
	case *types.Array:
		return e.containerTypeExpression(item, position)
	case *types.Map:
		return e.containerTypeExpression(item, position)
	case *types.Chan:
		return e.containerTypeExpression(item, position)
	case *types.TypeParam:
		return e.typeParameterExpression(item)
	case *types.Struct:
		return e.structTypeExpression(item, position)
	case *types.Interface:
		return e.interfaceTypeExpression(item, position)
	case *types.Union:
		return e.unionTypeExpression(item, position)
	case *types.Signature:
		return &ast.FuncType{
			Params:  e.tupleFields(item.Params(), item.Variadic(), position),
			Results: e.tupleFields(item.Results(), false, position),
		}
	}
	return nil
}

func (e *loweringEmitter) typeParameterExpression(item *types.TypeParam) ast.Expr {
	object := item.Obj()
	if alias := e.typeObjectAliases[object]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	return ast.NewIdent(object.Name())
}

func (e *loweringEmitter) containerTypeExpression(typ types.Type, position token.Pos) ast.Expr {
	switch item := typ.(type) {
	case *types.Pointer:
		return &ast.StarExpr{X: e.typeExpression(item.Elem(), position)}
	case *types.Slice:
		return &ast.ArrayType{Elt: e.typeExpression(item.Elem(), position)}
	case *types.Array:
		return &ast.ArrayType{
			Len: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(item.Len(), 10)},
			Elt: e.typeExpression(item.Elem(), position),
		}
	case *types.Map:
		return &ast.MapType{
			Key:   e.typeExpression(item.Key(), position),
			Value: e.typeExpression(item.Elem(), position),
		}
	case *types.Chan:
		direction := ast.SEND | ast.RECV
		if item.Dir() == types.SendOnly {
			direction = ast.SEND
		} else if item.Dir() == types.RecvOnly {
			direction = ast.RECV
		}
		return &ast.ChanType{Dir: direction, Value: e.typeExpression(item.Elem(), position)}
	}
	return nil
}

func (e *loweringEmitter) basicTypeExpression(item *types.Basic, position token.Pos) ast.Expr {
	if item.Info()&types.IsUntyped != 0 {
		return e.typeExpression(types.Default(item), position)
	}
	if alias := e.typeObjectAliases[types.Universe.Lookup(item.Name())]; alias != nil {
		return ast.NewIdent(alias.Name)
	}
	if item.Kind() == types.UnsafePointer {
		qualifier := e.unit.ownerQualifier(e.source.File, types.Unsafe)
		return e.unit.generatedObject(qualifier, types.Unsafe.Path(), "Pointer", position)
	}
	return e.unit.generatedUniverse(item.Name(), position)
}

func (e *loweringEmitter) namedTypeExpression(item *types.Named, position token.Pos) ast.Expr {
	object := item.Obj()
	var expression ast.Expr
	if alias := e.typeObjectAliases[object]; alias != nil {
		expression = ast.NewIdent(alias.Name)
	} else if object.Pkg() == e.unit.typed && object.Parent() != e.unit.typed.Scope() {
		expression = ast.NewIdent(object.Name())
	} else {
		qualifier := e.typeOwnerQualifier(object.Pkg())
		expression = e.unit.generatedObject(
			qualifier, packagePath(object.Pkg()), object.Name(), position,
		)
	}
	arguments := item.TypeArgs()
	if arguments == nil || arguments.Len() == 0 {
		return expression
	}
	indices := make([]ast.Expr, 0, arguments.Len())
	for index := range arguments.Len() {
		indices = append(indices, e.typeExpression(arguments.At(index), position))
	}
	return &ast.IndexListExpr{X: expression, Indices: indices}
}

func (e *loweringEmitter) structTypeExpression(
	item *types.Struct,
	position token.Pos,
) ast.Expr {
	fields := make([]*ast.Field, 0, item.NumFields())
	for index := range item.NumFields() {
		field := item.Field(index)
		var names []*ast.Ident
		if !field.Embedded() {
			names = []*ast.Ident{ast.NewIdent(field.Name())}
		}
		var tag *ast.BasicLit
		if text := item.Tag(index); text != "" {
			tag = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(text)}
		}
		fields = append(fields, &ast.Field{
			Names: names, Type: e.typeExpression(field.Type(), position), Tag: tag,
		})
	}
	return &ast.StructType{Fields: &ast.FieldList{List: fields}}
}

func (e *loweringEmitter) interfaceTypeExpression(
	item *types.Interface,
	position token.Pos,
) ast.Expr {
	fields := make([]*ast.Field, 0, item.NumExplicitMethods()+item.NumEmbeddeds())
	for index := range item.NumExplicitMethods() {
		method := item.ExplicitMethod(index)
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(method.Name())},
			Type:  e.typeExpression(method.Type(), position),
		})
	}
	for index := range item.NumEmbeddeds() {
		fields = append(fields, &ast.Field{
			Type: e.typeExpression(item.EmbeddedType(index), position),
		})
	}
	return &ast.InterfaceType{Methods: &ast.FieldList{List: fields}}
}

func (e *loweringEmitter) unionTypeExpression(item *types.Union, position token.Pos) ast.Expr {
	var expression ast.Expr
	for index := range item.Len() {
		term := item.Term(index)
		termExpression := e.typeExpression(term.Type(), position)
		if term.Tilde() {
			termExpression = &ast.UnaryExpr{Op: token.TILDE, X: termExpression}
		}
		if expression == nil {
			expression = termExpression
			continue
		}
		expression = &ast.BinaryExpr{X: expression, Op: token.OR, Y: termExpression}
	}
	return expression
}

func (e *loweringEmitter) tupleFields(
	tuple *types.Tuple,
	variadic bool,
	position token.Pos,
) *ast.FieldList {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}
	fields := make([]*ast.Field, 0, tuple.Len())
	for index := range tuple.Len() {
		typ := e.typeExpression(tuple.At(index).Type(), position)
		if variadic && index == tuple.Len()-1 {
			if slice, ok := typ.(*ast.ArrayType); ok && slice.Len == nil {
				typ = &ast.Ellipsis{Elt: slice.Elt}
			}
		}
		fields = append(fields, &ast.Field{Type: typ})
	}
	return &ast.FieldList{List: fields}
}

func packagePath(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}

func (e *loweringEmitter) expressionResults(
	plan *plannedExpression,
	block *ast.BlockStmt,
) []ast.Expr {
	if plan != nil && plan.resultCount > 1 && plan.work == nil {
		return []ast.Expr{e.expression(plan, block)}
	}
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

func (e *loweringEmitter) errorReturn(
	operation *plannedOperation,
	output *ast.BlockStmt,
) ast.Stmt {
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
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{specification},
		}})
		results = append(results, name)
	}
	returnedError := ast.Expr(e.valueName(operation.errorValue, "err"))
	if !operation.metadata.Transparent {
		returnedError = call(
			e.unit.generatedObject(e.formatQualifier(), "fmt", "Errorf", operation.metadata.Bang),
			&ast.BasicLit{
				Kind: token.STRING, Value: strconv.Quote(operation.metadata.Name + ": %w"),
			},
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

func oneStatement(statements []ast.Stmt) ast.Stmt {
	if len(statements) == 1 {
		return statements[0]
	}
	return &ast.BlockStmt{List: statements}
}

func positionGeneratedStatement(statement ast.Stmt, position token.Pos) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		positionGeneratedAssignment(node, position)
	case *ast.DeclStmt:
		if declaration, ok := node.Decl.(*ast.GenDecl); ok {
			positionGeneratedDeclaration(declaration, position)
		}
	case *ast.IfStmt:
		positionGeneratedIf(node, position)
	}
}

func positionGeneratedAssignment(statement *ast.AssignStmt, position token.Pos) {
	if statement.TokPos == token.NoPos {
		statement.TokPos = position
	}
	for _, expression := range statement.Lhs {
		positionGeneratedExpression(expression, position)
	}
	for _, expression := range statement.Rhs {
		positionGeneratedExpression(expression, position)
	}
}

func positionGeneratedDeclaration(declaration *ast.GenDecl, position token.Pos) {
	if declaration.TokPos == token.NoPos {
		declaration.TokPos = position
	}
	for _, specification := range declaration.Specs {
		value, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range value.Names {
			positionGeneratedExpression(name, position)
		}
		for _, expression := range value.Values {
			positionGeneratedExpression(expression, position)
		}
	}
}

func positionGeneratedIf(branch *ast.IfStmt, position token.Pos) {
	if branch.If == token.NoPos {
		branch.If = position
	}
	positionGeneratedExpression(branch.Cond, position)
	if branch.Body.Lbrace == token.NoPos {
		branch.Body.Lbrace = position
	}
	if branch.Body.Rbrace == token.NoPos {
		branch.Body.Rbrace = position
	}
	for _, statement := range branch.Body.List {
		if returned, ok := statement.(*ast.ReturnStmt); ok {
			if returned.Return == token.NoPos {
				returned.Return = position
			}
			for _, result := range returned.Results {
				positionGeneratedExpression(result, position)
			}
			continue
		}
		positionGeneratedStatement(statement, position)
	}
}

func positionGeneratedExpression(expression ast.Expr, position token.Pos) {
	switch node := expression.(type) {
	case *ast.Ident:
		if node.NamePos == token.NoPos {
			node.NamePos = position
		}
	case *ast.BasicLit:
		if node.ValuePos == token.NoPos {
			node.ValuePos = position
		}
	case *ast.BinaryExpr:
		if node.OpPos == token.NoPos {
			node.OpPos = position
		}
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Y, position)
	case *ast.CallExpr:
		if node.Lparen == token.NoPos {
			node.Lparen = position
		}
		if node.Rparen == token.NoPos {
			node.Rparen = position
		}
		positionGeneratedExpression(node.Fun, position)
		for _, argument := range node.Args {
			positionGeneratedExpression(argument, position)
		}
	case *ast.SelectorExpr:
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Sel, position)
	case *ast.CompositeLit:
		if node.Lbrace == token.NoPos {
			node.Lbrace = position
		}
		if node.Rbrace == token.NoPos {
			node.Rbrace = position
		}
	}
}
