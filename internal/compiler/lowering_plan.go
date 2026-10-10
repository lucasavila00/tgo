package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

type valueID int
type placeID int
type scopeID int
type targetID int

type plannedValue struct {
	id       valueID
	typ      types.Type
	position token.Pos
}

type plannedPlace struct {
	id       placeID
	typ      types.Type
	position token.Pos
	source   ast.Expr
	operands []*plannedExpression
	values   []plannedValue
	preserve []bool
	address  []bool
}

type plannedBlock struct {
	scope      scopeID
	operations []*plannedOperation
}

type plannedOperationKind uint8

const (
	planSourceStatement plannedOperationKind = iota
	planEvaluate
	planBind
	planStore
	planBranch
	planReturn
	planJump
	planBlockStatement
	planIfStatement
	planRangeStatement
	planForStatement
	planSwitchStatement
	planTypeSwitchStatement
	planSelectStatement
	planLabeledStatement
)

type plannedOperation struct {
	kind           plannedOperationKind
	source         ast.Stmt
	target         targetID
	expressions    []*plannedExpression
	places         []*plannedPlace
	init           *plannedBlock
	test           *plannedBlock
	body           *plannedBlock
	post           *plannedBlock
	otherwise      *plannedBlock
	cases          []*plannedBlock
	entry          []*plannedBlock
	selected       []*plannedBlock
	before         *plannedBlock
	inputs         []valueID
	outputs        []valueID
	errorValue     valueID
	metadata       *propagationSource
	operator       token.Token
	preferred      string
	communications []*plannedCommunication
	headerBindings []plannedHeaderBinding
	binding        *plannedOperation
	after          *plannedBlock
	declarations   []*plannedDeclaration
	failureCommas  []token.Pos
	labelHasGoto   bool
}

type plannedDeclaration struct {
	source      *ast.ValueSpec
	expressions []*plannedExpression
	before      *plannedBlock
	binding     *plannedOperation
	after       *plannedBlock
}

type plannedCommunication struct {
	source        ast.Stmt
	channel       *plannedExpression
	value         *plannedExpression
	channelValue  plannedValue
	sendValue     plannedValue
	receiveValues []plannedValue
	targets       []*plannedPlace
	left          []ast.Expr
	token         token.Token
}

type plannedHeaderBinding struct {
	target ast.Expr
	value  plannedValue
}

type plannedExpressionKind uint8

const (
	planRetainedExpression plannedExpressionKind = iota
	planCallExpression
	planBinaryExpression
	planIndexExpression
	planIndexListExpression
	planSelectorExpression
	planSliceExpression
	planParenExpression
	planStarExpression
	planUnaryExpression
	planTypeAssertExpression
	planCompositeExpression
	planKeyValueExpression
	planPropagationExpression
	planComprehensionExpression
)

type plannedExpression struct {
	kind          plannedExpressionKind
	source        ast.Expr
	typ           types.Type
	results       []plannedValue
	operands      []*plannedExpression
	propagation   *propagationSource
	comprehension *comprehensionSource
	function      *ast.FuncLit
	work          *plannedBlock
	before        *plannedBlock
	materialized  valueID
	resultNames   []*ast.Ident
	builtins      []*plannedBuiltin
	exact         *plannedExactComprehension
}

type plannedExactComprehension struct {
	makeCall   *ast.CallExpr
	outer      *ast.RangeStmt
	assignment *ast.AssignStmt
	appendCall *ast.CallExpr
	source     plannedValue
	position   token.Pos
}

type plannedBuiltin struct {
	call     *ast.CallExpr
	name     string
	position token.Pos
}

type functionLoweringPlan struct {
	function propagationFunction
	root     *plannedBlock
	values   []plannedValue
	places   []*plannedPlace
}

type loweringPlanBuilder struct {
	unit         *packageUnit
	source       *source
	function     propagationFunction
	plan         *functionLoweringPlan
	nextScope    scopeID
	nextTarget   targetID
	bindings     map[types.Object]types.Type
	currentScope scopeID
	gotos        map[string]bool
}

func buildFunctionLoweringPlan(
	unit *packageUnit,
	source *source,
	function propagationFunction,
) *functionLoweringPlan {
	plan := &functionLoweringPlan{function: function}
	builder := &loweringPlanBuilder{
		unit: unit, source: source, function: function, plan: plan,
		bindings: make(map[types.Object]types.Type),
		gotos:    functionGotoLabels(function.body),
	}
	plan.root = builder.block(function.body.List)
	return plan
}

func (b *loweringPlanBuilder) block(statements []ast.Stmt) *plannedBlock {
	outerBindings := b.bindings
	b.bindings = cloneTypeBindings(outerBindings)
	outerScope := b.currentScope
	b.nextScope++
	b.currentScope = b.nextScope
	block := &plannedBlock{scope: b.currentScope}
	for _, statement := range statements {
		beforeStatement := cloneTypeBindings(b.bindings)
		block.operations = append(block.operations, b.statement(statement))
		if statementOwnsScope(statement) {
			b.bindings = beforeStatement
		} else {
			b.recordBindings(statement)
		}
	}
	b.bindings = outerBindings
	b.currentScope = outerScope
	return block
}

func statementOwnsScope(statement ast.Stmt) bool {
	switch statement.(type) {
	case *ast.BlockStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
		*ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	}
	return false
}

func cloneTypeBindings(input map[types.Object]types.Type) map[types.Object]types.Type {
	result := make(map[types.Object]types.Type, len(input))
	for object, typ := range input {
		result[object] = typ
	}
	return result
}

// statement defines the execution regions for one source statement.
func (b *loweringPlanBuilder) statement(statement ast.Stmt) *plannedOperation {
	operation := &plannedOperation{source: statement, kind: planSourceStatement}
	switch node := statement.(type) {
	case *ast.BlockStmt:
		operation.kind = planBlockStatement
		operation.body = b.block(node.List)
	case *ast.IfStmt:
		operation.kind = planIfStatement
		operation.init = b.simpleBlock(node.Init)
		condition := b.expression(node.Cond)
		operation.test = b.plannedExpressionBlock(condition)
		operation.expressions = []*plannedExpression{condition}
		operation.body = b.block(node.Body.List)
		if node.Else != nil {
			operation.otherwise = b.block([]ast.Stmt{node.Else})
		}
	case *ast.RangeStmt:
		operation.kind = planRangeStatement
		operation.expressions = b.expressions(node.X)
		operation.body = b.block(node.Body.List)
		operation.places = b.assignmentPlaces(node.Key, node.Value)
		operation.target = b.target()
	case *ast.ForStmt:
		operation.kind = planForStatement
		operation.init, operation.headerBindings = b.loopInitializer(node.Init)
		if node.Cond != nil {
			condition := b.expression(node.Cond)
			operation.test = b.plannedExpressionBlock(condition)
			operation.expressions = []*plannedExpression{condition}
		}
		operation.body = b.block(node.Body.List)
		operation.post = b.simpleBlock(node.Post)
		operation.target = b.target()
	case *ast.SwitchStmt:
		operation.kind = planSwitchStatement
		operation.init = b.simpleBlock(node.Init)
		if node.Tag != nil {
			tag := b.expression(node.Tag)
			operation.test = b.plannedExpressionBlock(tag)
			operation.expressions = []*plannedExpression{tag}
		}
		operation.target = b.target()
		for _, item := range node.Body.List {
			clause := item.(*ast.CaseClause)
			operation.cases = append(operation.cases, b.block(clause.Body))
			for _, expression := range clause.List {
				caseExpression := b.expression(expression)
				operation.entry = append(operation.entry, b.plannedExpressionBlock(caseExpression))
				operation.expressions = append(operation.expressions, caseExpression)
			}
		}
	case *ast.TypeSwitchStmt:
		operation.kind = planTypeSwitchStatement
		operation.init = b.simpleBlock(node.Init)
		operation.test = b.simpleBlock(node.Assign)
		operation.target = b.target()
		for _, item := range node.Body.List {
			operation.cases = append(operation.cases, b.block(item.(*ast.CaseClause).Body))
		}
	case *ast.SelectStmt:
		operation.kind = planSelectStatement
		operation.target = b.target()
		for _, item := range node.Body.List {
			clause := item.(*ast.CommClause)
			operation.communications = append(operation.communications, b.communication(clause.Comm))
			operation.entry = append(operation.entry, b.selectEntry(clause.Comm))
			operation.selected = append(operation.selected, b.selectSelected(clause.Comm))
			operation.cases = append(operation.cases, b.block(clause.Body))
		}
	case *ast.LabeledStmt:
		operation.kind = planLabeledStatement
		operation.target = b.target()
		operation.body = b.block([]ast.Stmt{node.Stmt})
		operation.labelHasGoto = b.gotos[node.Label.Name]
	default:
		if declaration, ok := statement.(*ast.DeclStmt); ok {
			operation.declarations = b.declarationPlans(declaration)
			for _, specification := range operation.declarations {
				operation.expressions = append(operation.expressions, specification.expressions...)
			}
		} else {
			operation.expressions = b.statementExpressions(statement)
			operation.before = b.orderExpressions(operation.expressions)
		}
		if assignment, ok := statement.(*ast.AssignStmt); ok {
			operation.places = b.assignmentPlaces(assignment.Lhs...)
		}
		b.directStatementBinding(operation)
	}
	if returned, ok := statement.(*ast.ReturnStmt); ok {
		operation.failureCommas = append(operation.failureCommas, b.source.FailureReturns[returned]...)
	}
	return operation
}

func (b *loweringPlanBuilder) declarationPlans(
	statement *ast.DeclStmt,
) []*plannedDeclaration {
	general, ok := statement.Decl.(*ast.GenDecl)
	if !ok {
		return nil
	}
	result := []*plannedDeclaration(nil)
	for _, item := range general.Specs {
		value, ok := item.(*ast.ValueSpec)
		if !ok {
			continue
		}
		planned := &plannedDeclaration{
			source: value, expressions: b.expressionList(value.Values),
		}
		planned.before = b.orderExpressions(planned.expressions)
		temporarySource := &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: general.Tok, Specs: []ast.Spec{value},
		}}
		temporary := &plannedOperation{
			source: temporarySource, expressions: planned.expressions,
		}
		b.directStatementBinding(temporary)
		planned.binding, planned.after = temporary.binding, temporary.after
		result = append(result, planned)
		targets := make([]ast.Expr, len(value.Names))
		for index, name := range value.Names {
			targets[index] = name
		}
		b.bindExpressions(targets, value.Values)
	}
	return result
}

func (b *loweringPlanBuilder) directStatementBinding(operation *plannedOperation) {
	if len(operation.expressions) != 1 {
		return
	}
	expression := operation.expressions[0]
	if expression.work == nil || len(expression.work.operations) < 2 {
		return
	}
	binding := expression.work.operations[0]
	if binding.kind != planBind || len(binding.outputs) != len(expression.results)+1 {
		return
	}
	targets := 0
	switch node := operation.source.(type) {
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE {
			return
		}
		targets = len(node.Lhs)
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR || len(declaration.Specs) != 1 {
			return
		}
		value, ok := declaration.Specs[0].(*ast.ValueSpec)
		if !ok || value.Type != nil || len(value.Values) != 1 {
			return
		}
		targets = len(value.Names)
	default:
		return
	}
	if targets != len(expression.results) {
		return
	}
	operation.binding = binding
	operation.after = &plannedBlock{
		scope:      expression.work.scope,
		operations: expression.work.operations[1:],
	}
}

func (b *loweringPlanBuilder) communication(statement ast.Stmt) *plannedCommunication {
	communication := &plannedCommunication{source: statement}
	var channel ast.Expr
	switch node := statement.(type) {
	case *ast.SendStmt:
		channel = node.Chan
		communication.value = b.expression(node.Value)
	case *ast.ExprStmt:
		if receive, ok := node.X.(*ast.UnaryExpr); ok && receive.Op == token.ARROW {
			channel = receive.X
		}
	case *ast.AssignStmt:
		communication.left = node.Lhs
		communication.token = node.Tok
		communication.targets = b.assignmentPlaces(node.Lhs...)
		if len(node.Rhs) == 1 {
			if receive, ok := node.Rhs[0].(*ast.UnaryExpr); ok && receive.Op == token.ARROW {
				channel = receive.X
			}
		}
	}
	if channel == nil {
		return communication
	}
	communication.channel = b.expression(channel)
	communication.channelValue = b.newValue(b.expressionType(channel), channel.Pos())
	channelType := b.expressionType(channel)
	if channelType == nil {
		return communication
	}
	if named, ok := types.Unalias(channelType).(*types.Named); ok {
		channelType = named.Underlying()
	}
	if channelType, ok := channelType.(*types.Chan); ok {
		if communication.value != nil {
			communication.sendValue = b.newValue(channelType.Elem(), communication.value.source.Pos())
		} else if len(communication.left) != 0 {
			communication.receiveValues = append(communication.receiveValues,
				b.newValue(channelType.Elem(), channel.Pos()))
			if len(communication.left) == 2 {
				communication.receiveValues = append(communication.receiveValues,
					b.newValue(types.Typ[types.Bool], channel.Pos()))
			}
		}
	}
	return communication
}

func (b *loweringPlanBuilder) newValue(typ types.Type, position token.Pos) plannedValue {
	value := plannedValue{id: valueID(len(b.plan.values) + 1), typ: typ, position: position}
	b.plan.values = append(b.plan.values, value)
	return value
}

func (b *loweringPlanBuilder) orderExpressions(
	expressions []*plannedExpression,
) *plannedBlock {
	block := &plannedBlock{scope: b.currentScope}
	for index, expression := range expressions {
		laterWork := false
		for _, later := range expressions[index+1:] {
			if plannedExpressionHasWork(later) {
				laterWork = true
				break
			}
		}
		if !laterWork || !b.canMaterialize(expression) {
			continue
		}
		value := plannedValue{
			id: valueID(len(b.plan.values) + 1), typ: expression.typ,
			position: expression.source.Pos(),
		}
		b.plan.values = append(b.plan.values, value)
		expression.materialized = value.id
		block.operations = append(block.operations, &plannedOperation{
			kind: planEvaluate, expressions: []*plannedExpression{expression},
			outputs: []valueID{value.id},
		})
	}
	if len(block.operations) == 0 {
		return nil
	}
	return block
}

func (b *loweringPlanBuilder) canMaterialize(expression *plannedExpression) bool {
	if expression == nil || !validPlannedType(expression.typ) {
		return false
	}
	if _, tuple := expression.typ.(*types.Tuple); tuple {
		return false
	}
	value, ok := b.unit.info.Types[expression.source]
	if ok && value.Value != nil {
		return false
	}
	switch expression.source.(type) {
	case *ast.Ident:
		return false
	case *ast.BasicLit:
		return false
	}
	return true
}

func (b *loweringPlanBuilder) target() targetID {
	b.nextTarget++
	return b.nextTarget
}

func (b *loweringPlanBuilder) simpleBlock(statement ast.Stmt) *plannedBlock {
	if statement == nil {
		return nil
	}
	block := &plannedBlock{scope: b.currentScope}
	block.operations = append(block.operations, b.statement(statement))
	b.recordBindings(statement)
	return block
}

func (b *loweringPlanBuilder) loopInitializer(
	statement ast.Stmt,
) (*plannedBlock, []plannedHeaderBinding) {
	if statement == nil {
		return nil, nil
	}
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE {
		return b.simpleBlock(statement), nil
	}
	expressions := b.expressionList(assignment.Rhs)
	hasWork := false
	for _, expression := range expressions {
		hasWork = hasWork || plannedExpressionHasWork(expression)
	}
	if !hasWork {
		return b.simpleBlock(statement), nil
	}
	evaluation := &plannedBlock{scope: b.currentScope}
	values := []plannedValue(nil)
	for _, expression := range expressions {
		outputs := make([]valueID, 0, len(expression.results))
		for _, value := range expression.results {
			outputs = append(outputs, value.id)
			values = append(values, value)
		}
		if expression.work == nil && len(expression.results) == 1 {
			expression.materialized = expression.results[0].id
		}
		evaluation.operations = append(evaluation.operations, &plannedOperation{
			kind: planEvaluate, expressions: []*plannedExpression{expression}, outputs: outputs,
		})
	}
	if len(values) != len(assignment.Lhs) {
		return b.simpleBlock(statement), nil
	}
	bindings := make([]plannedHeaderBinding, 0, len(values))
	for index, value := range values {
		bindings = append(bindings, plannedHeaderBinding{
			target: assignment.Lhs[index], value: value,
		})
		b.bindTarget(assignment.Lhs[index], value.typ)
	}
	return evaluation, bindings
}

func (b *loweringPlanBuilder) expressionBlock(expression ast.Expr) *plannedBlock {
	if expression == nil {
		return nil
	}
	return b.plannedExpressionBlock(b.expression(expression))
}

func (b *loweringPlanBuilder) plannedExpressionBlock(
	expression *plannedExpression,
) *plannedBlock {
	return &plannedBlock{scope: b.currentScope, operations: []*plannedOperation{{
		kind: planEvaluate, expressions: []*plannedExpression{expression},
	}}}
}

func (b *loweringPlanBuilder) statementExpressions(statement ast.Stmt) []*plannedExpression {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		return b.expressionList(node.Rhs)
	case *ast.IncDecStmt:
		return b.expressions(node.X)
	case *ast.ExprStmt:
		return b.expressions(node.X)
	case *ast.ReturnStmt:
		return b.expressionList(node.Results)
	case *ast.SendStmt:
		return b.expressionList([]ast.Expr{node.Chan, node.Value})
	case *ast.GoStmt:
		return b.expressions(node.Call)
	case *ast.DeferStmt:
		return b.expressions(node.Call)
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok {
			return nil
		}
		result := []*plannedExpression(nil)
		for _, specification := range declaration.Specs {
			if value, ok := specification.(*ast.ValueSpec); ok {
				result = append(result, b.expressionList(value.Values)...)
			}
		}
		return result
	}
	return nil
}

func (b *loweringPlanBuilder) expressionList(input []ast.Expr) []*plannedExpression {
	result := make([]*plannedExpression, 0, len(input))
	for _, expression := range input {
		result = append(result, b.expressions(expression)...)
	}
	return result
}

func (b *loweringPlanBuilder) expressions(expression ast.Expr) []*plannedExpression {
	if expression == nil {
		return nil
	}
	return []*plannedExpression{b.expression(expression)}
}

func (b *loweringPlanBuilder) expression(expression ast.Expr) *plannedExpression {
	result := &plannedExpression{
		kind: planRetainedExpression, source: expression, typ: b.expressionType(expression),
	}
	if metadata, function, ok := comprehensionMarker(b.source, expression); ok {
		result.kind = planComprehensionExpression
		result.comprehension = &metadata
		result.function = function
		if function != nil && function.Body != nil {
			result.work = b.block(function.Body.List)
			if count := len(result.work.operations); count != 0 {
				if _, ok := result.work.operations[count-1].source.(*ast.ReturnStmt); ok {
					result.work.operations = result.work.operations[:count-1]
				}
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if ok && identifier.Name == metadata.Result {
					result.resultNames = append(result.resultNames, identifier)
				}
				return true
			})
			if makeCall, outer, valid := b.comprehensionParts(function.Body); valid {
				if !validPlannedType(result.typ) && len(makeCall.Args) != 0 {
					result.typ = b.expressionType(makeCall.Args[0])
				}
				result.builtins = append(result.builtins, &plannedBuiltin{
					call: makeCall, name: "make", position: metadata.Position,
				})
				terminal := comprehensionTerminal(outer.Body)
				if len(terminal.List) == 1 {
					if assignment, ok := terminal.List[0].(*ast.AssignStmt); ok &&
						len(assignment.Rhs) == 1 {
						if appendCall, ok := assignment.Rhs[0].(*ast.CallExpr); ok {
							result.builtins = append(result.builtins, &plannedBuiltin{
								call: appendCall, name: "append", position: metadata.Position,
							})
							if len(makeCall.Args) == 2 && len(function.Body.List) == 3 &&
								len(outer.Body.List) == 1 &&
								underlyingSlice(b.expressionType(outer.X)) != nil &&
								underlyingSlice(b.expressionType(makeCall.Args[0])) != nil {
								for _, operation := range result.work.operations {
									if operation.source != outer || len(operation.expressions) != 1 {
										continue
									}
									sourcePlan := operation.expressions[0]
									sourceValue := b.newValue(sourcePlan.typ, outer.X.Pos())
									sourcePlan.materialized = sourceValue.id
									result.work.operations = append([]*plannedOperation{{
										kind: planEvaluate, expressions: []*plannedExpression{sourcePlan},
										outputs: []valueID{sourceValue.id}, preferred: "source",
									}}, result.work.operations...)
									result.exact = &plannedExactComprehension{
										makeCall: makeCall, outer: outer, assignment: assignment,
										appendCall: appendCall, source: sourceValue,
										position: metadata.Position,
									}
									break
								}
							}
						}
					}
				}
			}
		}
		result.addResult(b, result.typ, expression.Pos())
		return result
	}
	if metadata, ok := propagationMarker(b.source, expression); ok {
		result.kind = planPropagationExpression
		result.propagation = &metadata
		result.work = &plannedBlock{scope: b.currentScope}
		marker := expression.(*ast.CallExpr)
		if call, valid := unwrappedCompilerCall(marker.Args[0]); valid {
			callPlan := b.expression(call)
			result.operands = append(result.operands, callPlan)
			if signature := b.callSignature(call.Fun); signature != nil {
				for index := 0; index < signature.Results().Len(); index++ {
					result.addResult(b, signature.Results().At(index).Type(), expression.Pos())
				}
				if len(result.results) != 0 {
					errorResult := result.results[len(result.results)-1]
					result.results = result.results[:len(result.results)-1]
					outputs := make([]valueID, 0, len(result.results)+1)
					for _, value := range result.results {
						outputs = append(outputs, value.id)
					}
					outputs = append(outputs, errorResult.id)
					result.work.operations = append(result.work.operations,
						&plannedOperation{
							kind: planBind, expressions: []*plannedExpression{callPlan}, outputs: outputs,
							metadata: &metadata,
						},
						&plannedOperation{
							kind: planBranch, errorValue: errorResult.id, metadata: &metadata,
							body: &plannedBlock{scope: b.currentScope, operations: []*plannedOperation{{
								kind: planReturn, errorValue: errorResult.id, metadata: &metadata,
							}}},
						},
					)
				}
			}
		}
		return result
	}
	var operands []ast.Expr
	switch node := expression.(type) {
	case *ast.CallExpr:
		result.kind = planCallExpression
		operands = append([]ast.Expr{node.Fun}, node.Args...)
	case *ast.BinaryExpr:
		result.kind = planBinaryExpression
		if node.Op == token.LAND || node.Op == token.LOR {
			left := b.expression(node.X)
			right := b.expression(node.Y)
			result.operands = []*plannedExpression{left, right}
			result.addResult(b, result.typ, expression.Pos())
			if len(result.results) != 0 {
				output := result.results[0].id
				result.work = &plannedBlock{
					scope: b.currentScope,
					operations: []*plannedOperation{
						{kind: planBind, expressions: []*plannedExpression{left}, outputs: []valueID{output}},
						{
							kind: planBranch, inputs: []valueID{output}, operator: node.Op,
							body: &plannedBlock{scope: b.currentScope, operations: []*plannedOperation{{
								kind: planStore, expressions: []*plannedExpression{right}, inputs: []valueID{output},
							}}},
						},
					},
				}
			}
			return result
		}
		operands = []ast.Expr{node.X, node.Y}
	case *ast.IndexExpr:
		result.kind = planIndexExpression
		operands = []ast.Expr{node.X, node.Index}
	case *ast.IndexListExpr:
		result.kind = planIndexListExpression
		operands = append([]ast.Expr{node.X}, node.Indices...)
	case *ast.SelectorExpr:
		result.kind = planSelectorExpression
		operands = []ast.Expr{node.X}
	case *ast.SliceExpr:
		result.kind = planSliceExpression
		operands = []ast.Expr{node.X, node.Low, node.High, node.Max}
	case *ast.ParenExpr:
		result.kind = planParenExpression
		operands = []ast.Expr{node.X}
	case *ast.StarExpr:
		result.kind = planStarExpression
		operands = []ast.Expr{node.X}
	case *ast.UnaryExpr:
		result.kind = planUnaryExpression
		operands = []ast.Expr{node.X}
	case *ast.TypeAssertExpr:
		result.kind = planTypeAssertExpression
		operands = []ast.Expr{node.X}
	case *ast.CompositeLit:
		result.kind = planCompositeExpression
		operands = node.Elts
	case *ast.KeyValueExpr:
		result.kind = planKeyValueExpression
		operands = []ast.Expr{node.Key, node.Value}
	case *ast.FuncLit:
		return result
	}
	for _, operand := range operands {
		if operand != nil {
			result.operands = append(result.operands, b.expression(operand))
		}
	}
	result.before = b.orderOperands(result.operands)
	result.addResult(b, result.typ, expression.Pos())
	return result
}

func (b *loweringPlanBuilder) orderOperands(
	operands []*plannedExpression,
) *plannedBlock {
	block := &plannedBlock{scope: b.currentScope}
	for index, operand := range operands {
		laterWork := false
		for _, later := range operands[index+1:] {
			if plannedExpressionHasWork(later) {
				laterWork = true
				break
			}
		}
		operandWork := plannedExpressionHasWork(operand)
		if !operandWork && (!laterWork || !b.canMaterialize(operand)) {
			continue
		}
		if len(operand.results) != 1 {
			continue
		}
		operand.materialized = operand.results[0].id
		block.operations = append(block.operations, &plannedOperation{
			kind: planEvaluate, expressions: []*plannedExpression{operand},
			outputs: []valueID{operand.results[0].id},
		})
	}
	if len(block.operations) == 0 {
		return nil
	}
	return block
}

func (b *loweringPlanBuilder) comprehensionParts(
	body *ast.BlockStmt,
) (*ast.CallExpr, *ast.RangeStmt, bool) {
	if len(body.List) < 3 {
		return nil, nil, false
	}
	initialization, ok := body.List[0].(*ast.AssignStmt)
	if !ok || len(initialization.Rhs) != 1 {
		return nil, nil, false
	}
	makeCall, ok := initialization.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	outer, ok := body.List[1].(*ast.RangeStmt)
	return makeCall, outer, ok
}

func (e *plannedExpression) addResult(
	b *loweringPlanBuilder,
	typ types.Type,
	position token.Pos,
) {
	if typ == nil {
		return
	}
	if tuple, ok := typ.(*types.Tuple); ok {
		for index := range tuple.Len() {
			e.addResult(b, tuple.At(index).Type(), position)
		}
		return
	}
	value := plannedValue{id: valueID(len(b.plan.values) + 1), typ: typ, position: position}
	b.plan.values = append(b.plan.values, value)
	e.results = append(e.results, value)
}

func (b *loweringPlanBuilder) assignmentPlaces(expressions ...ast.Expr) []*plannedPlace {
	result := make([]*plannedPlace, 0, len(expressions))
	for _, expression := range expressions {
		if expression == nil {
			continue
		}
		place := &plannedPlace{
			id: placeID(len(b.plan.places) + 1), typ: b.expressionType(expression),
			position: expression.Pos(), source: expression, operands: b.placeOperands(expression),
		}
		for index, operand := range place.operands {
			address := false
			if indexed, ok := expression.(*ast.IndexExpr); ok && index == 0 {
				address = admitsArray(b.expressionType(indexed.X)) &&
					b.unit.info.Types[indexed.X].Addressable()
			}
			if selector, ok := expression.(*ast.SelectorExpr); ok && index == 0 {
				_, pointer := types.Unalias(b.expressionType(selector.X)).(*types.Pointer)
				address = !pointer && b.unit.info.Types[selector.X].Addressable()
			}
			place.address = append(place.address, address)
			place.preserve = append(place.preserve, false)
			if operand.typ == nil {
				place.values = append(place.values, plannedValue{})
				continue
			}
			typ := operand.typ
			if address {
				typ = types.NewPointer(typ)
			}
			place.values = append(place.values, b.newValue(typ, operand.source.Pos()))
		}
		b.plan.places = append(b.plan.places, place)
		result = append(result, place)
	}
	return result
}

// recordBindings carries types from propagation results by source object.
func (b *loweringPlanBuilder) recordBindings(statement ast.Stmt) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE {
			return
		}
		b.bindExpressions(node.Lhs, node.Rhs)
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR {
			return
		}
		for _, specification := range declaration.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			targets := make([]ast.Expr, len(value.Names))
			for index, name := range value.Names {
				targets[index] = name
			}
			b.bindExpressions(targets, value.Values)
		}
	}
}

func (b *loweringPlanBuilder) bindExpressions(targets []ast.Expr, values []ast.Expr) {
	if len(values) == 1 {
		planned := b.expression(values[0])
		if len(planned.results) == len(targets) {
			for index, target := range targets {
				b.bindTarget(target, planned.results[index].typ)
			}
			return
		}
	}
	if len(values) != len(targets) {
		return
	}
	for index, target := range targets {
		b.bindTarget(target, b.expressionType(values[index]))
	}
}

func (b *loweringPlanBuilder) bindTarget(target ast.Expr, typ types.Type) {
	identifier, ok := target.(*ast.Ident)
	if !ok || identifier.Name == "_" || typ == nil {
		return
	}
	if object := b.unit.info.Defs[identifier]; object != nil {
		b.bindings[object] = typ
	}
}

func (b *loweringPlanBuilder) expressionType(expression ast.Expr) types.Type {
	if expression == nil {
		return nil
	}
	if typ := b.unit.info.TypeOf(expression); validPlannedType(typ) {
		return typ
	}
	switch node := expression.(type) {
	case *ast.ParenExpr:
		return b.expressionType(node.X)
	case *ast.Ident:
		return b.bindings[b.unit.info.ObjectOf(node)]
	case *ast.IndexExpr:
		container := b.expressionType(node.X)
		if container == nil {
			return nil
		}
		switch value := types.Unalias(container).Underlying().(type) {
		case *types.Array:
			return value.Elem()
		case *types.Slice:
			return value.Elem()
		case *types.Map:
			return value.Elem()
		case *types.Pointer:
			if array, ok := types.Unalias(value.Elem()).Underlying().(*types.Array); ok {
				return array.Elem()
			}
		}
	case *ast.SelectorExpr:
		receiver := b.expressionType(node.X)
		if receiver != nil {
			object, _, _ := types.LookupFieldOrMethod(receiver, true, b.unit.typed, node.Sel.Name)
			if object != nil {
				return object.Type()
			}
		}
	case *ast.CallExpr:
		if signature := b.callSignature(node.Fun); signature != nil {
			if signature.Results().Len() == 1 {
				return signature.Results().At(0).Type()
			}
			return signature.Results()
		}
	case *ast.BinaryExpr:
		switch node.Op {
		case token.LAND, token.LOR, token.EQL, token.NEQ,
			token.LSS, token.LEQ, token.GTR, token.GEQ:
			return types.Typ[types.Bool]
		}
	}
	return nil
}

func validPlannedType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if basic, ok := types.Unalias(typ).(*types.Basic); ok {
		return basic.Kind() != types.Invalid
	}
	return true
}

func (b *loweringPlanBuilder) callSignature(function ast.Expr) *types.Signature {
	signature, _ := types.Unalias(b.expressionType(function)).(*types.Signature)
	return signature
}

func (b *loweringPlanBuilder) placeOperands(expression ast.Expr) []*plannedExpression {
	switch node := expression.(type) {
	case *ast.ParenExpr:
		return b.placeOperands(node.X)
	case *ast.StarExpr:
		return b.expressions(node.X)
	case *ast.SelectorExpr:
		return b.expressions(node.X)
	case *ast.IndexExpr:
		return b.expressionList([]ast.Expr{node.X, node.Index})
	default:
		return nil
	}
}

// selectEntry records operands that Go evaluates when it enters a select.
func (b *loweringPlanBuilder) selectEntry(statement ast.Stmt) *plannedBlock {
	if statement == nil {
		return nil
	}
	b.nextScope++
	block := &plannedBlock{scope: b.nextScope}
	operation := &plannedOperation{kind: planSourceStatement, source: statement}
	switch node := statement.(type) {
	case *ast.SendStmt:
		operation.expressions = b.expressionList([]ast.Expr{node.Chan, node.Value})
	case *ast.ExprStmt:
		if unary, ok := node.X.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
			operation.expressions = b.expressions(unary.X)
		}
	case *ast.AssignStmt:
		operation.expressions = b.expressionList(node.Rhs)
	}
	block.operations = append(block.operations, operation)
	return block
}

// selectSelected records receive targets that Go evaluates only after selection.
func (b *loweringPlanBuilder) selectSelected(statement ast.Stmt) *plannedBlock {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	b.nextScope++
	return &plannedBlock{
		scope: b.nextScope,
		operations: []*plannedOperation{{
			kind: planSourceStatement, source: statement,
			places: b.assignmentPlaces(assignment.Lhs...),
		}},
	}
}
