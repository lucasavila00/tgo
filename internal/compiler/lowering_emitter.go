package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

type loweringEmitter struct {
	unit                *packageUnit
	source              *source
	plan                *functionLoweringPlan
	names               map[string]bool
	values              map[valueID]*ast.Ident
	targets             map[targetID]*ast.Ident
	typeAliases         map[*ast.BlockStmt]map[types.Type]*ast.Ident
	renamedTypeBlockers map[types.Object]string
	typeDefinitionSites map[*ast.Ident]emittedTypeDefinition
	typeObjectAliases   map[types.Object]*ast.Ident
	forcedQualifiers    map[*types.Package]string
	packageQualifiers   map[string]string
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
	if source.LoweringPackageQualifiers == nil {
		source.LoweringPackageQualifiers = make(map[string]string)
	}
	emitter := &loweringEmitter{
		unit: unit, source: source, plan: plan, names: plan.function.names,
		values: make(map[valueID]*ast.Ident), targets: make(map[targetID]*ast.Ident),
		typeAliases:         make(map[*ast.BlockStmt]map[types.Type]*ast.Ident),
		renamedTypeBlockers: make(map[types.Object]string),
		typeDefinitionSites: make(map[*ast.Ident]emittedTypeDefinition),
		typeObjectAliases:   source.LoweringTypeAliases,
		forcedQualifiers:    make(map[*types.Package]string),
		packageQualifiers:   source.LoweringPackageQualifiers,
	}
	for _, alias := range source.LoweringTypeAliases {
		emitter.names[alias.Name] = true
	}
	for _, qualifier := range source.LoweringPackageQualifiers {
		emitter.names[qualifier] = true
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
		return e.preparedPlaceExpression(plan.place)
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
		X: value, Op: token.EQL, Y: e.untypedTrueExpression(),
	}
}

func (e *loweringEmitter) untypedBooleanAdapter(value ast.Expr) ast.Expr {
	return &ast.BinaryExpr{X: value, Op: token.EQL, Y: e.untypedTrueExpression()}
}

func (e *loweringEmitter) untypedTrueExpression() ast.Expr {
	return &ast.BinaryExpr{
		X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: token.EQL,
		Y: &ast.BasicLit{Kind: token.INT, Value: "0"},
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
	if e.emitAssignmentValueOperation(operation, output) {
		return
	}
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
	case planBooleanConvert:
		e.emitBooleanConversion(operation, output)
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
	}
}

func (e *loweringEmitter) emitBooleanConversion(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	source := e.valueName(operation.inputs[0], "result")
	value := e.plannedValue(operation.outputs[0])
	e.emitTypedExpressionBind(value, e.untypedBooleanAdapter(source), false, output)
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
		results := e.expressionResults(expression, output)
		expression.materialized = materialized
		if evaluationOutputsMatchResults(operation.outputs, expression.results) {
			return
		}
		e.emitEvaluationOutputs(operation.outputs, results, output)
		return
	}
	value := e.expression(expression, output)
	expression.materialized = materialized
	if len(operation.outputs) == 0 {
		return
	}
	if len(operation.outputs) > 1 {
		e.emitEvaluationOutputs(operation.outputs, []ast.Expr{value}, output)
		return
	}
	planned := e.plannedValue(operation.outputs[0])
	if operation.preferred != "" && e.values[planned.id] == nil {
		e.values[planned.id] = e.freshName(operation.preferred)
	}
	planned.explicit = expression.contextual
	e.emitTypedExpressionBind(planned, value, planned.explicit, output)
}

func evaluationOutputsMatchResults(outputs []valueID, results []plannedValue) bool {
	if len(outputs) != len(results) {
		return false
	}
	for index, output := range outputs {
		if output != results[index].id {
			return false
		}
	}
	return true
}

func (e *loweringEmitter) emitEvaluationOutputs(
	outputs []valueID,
	results []ast.Expr,
	block *ast.BlockStmt,
) {
	left := make([]ast.Expr, 0, len(outputs))
	for _, output := range outputs {
		left = append(left, e.valueName(output, "result"))
	}
	block.List = append(block.List, &ast.AssignStmt{
		Lhs: left, Tok: token.DEFINE, Rhs: results,
	})
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
