package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *loweringPlanBuilder) statement(statement ast.Stmt) *plannedOperation {
	operation := &plannedOperation{
		source: statement, kind: planSourceStatement,
		sourceScope: b.sourceScope(statement.Pos()),
	}
	switch node := statement.(type) {
	case *ast.BlockStmt:
		b.planBlockStatement(operation, node)
	case *ast.IfStmt:
		b.planIfStatement(operation, node)
	case *ast.RangeStmt:
		b.planRangeStatement(operation, node)
	case *ast.ForStmt:
		b.planForStatement(operation, node)
	case *ast.SwitchStmt:
		b.planSwitchStatement(operation, node)
	case *ast.TypeSwitchStmt:
		b.planTypeSwitchStatement(operation, node)
	case *ast.SelectStmt:
		b.planSelectStatement(operation, node)
	case *ast.LabeledStmt:
		b.planLabeledStatement(operation, node)
	case *ast.BranchStmt:
		b.planBranchStatement(operation, node)
	default:
		b.planDirectStatement(operation, statement)
	}
	if returned, ok := statement.(*ast.ReturnStmt); ok {
		operation.failureCommas = append(
			operation.failureCommas, b.source.FailureReturns[returned]...,
		)
	}
	return operation
}

func (b *loweringPlanBuilder) planBlockStatement(operation *plannedOperation, node *ast.BlockStmt) {
	operation.kind = planBlockStatement
	operation.body = b.block(node.List)
}

func (b *loweringPlanBuilder) planIfStatement(operation *plannedOperation, node *ast.IfStmt) {
	operation.kind = planIfStatement
	operation.init = b.simpleBlock(node.Init)
	condition := b.expression(node.Cond)
	operation.test = b.plannedExpressionBlock(condition)
	operation.expressions = []*plannedExpression{condition}
	operation.body = b.block(node.Body.List)
	if node.Else != nil {
		operation.otherwise = b.block([]ast.Stmt{node.Else})
	}
}

func (b *loweringPlanBuilder) planRangeStatement(operation *plannedOperation, node *ast.RangeStmt) {
	operation.kind = planRangeStatement
	operation.target = b.target()
	operation.expressions = b.expressions(node.X)
	b.breakTargets = append(b.breakTargets, operation.target)
	b.loopTargets = append(b.loopTargets, operation.target)
	operation.body = b.block(node.Body.List)
	b.breakTargets = b.breakTargets[:len(b.breakTargets)-1]
	b.loopTargets = b.loopTargets[:len(b.loopTargets)-1]
	operation.places = b.assignmentPlaces(node.Key, node.Value)
}

func (b *loweringPlanBuilder) planForStatement(operation *plannedOperation, node *ast.ForStmt) {
	operation.kind = planForStatement
	operation.target = b.target()
	operation.init, operation.headerBindings = b.loopInitializer(node.Init)
	if node.Cond != nil {
		condition := b.expression(node.Cond)
		operation.test = b.plannedExpressionBlock(condition)
		operation.expressions = []*plannedExpression{condition}
	}
	b.breakTargets = append(b.breakTargets, operation.target)
	b.loopTargets = append(b.loopTargets, operation.target)
	operation.body = b.block(node.Body.List)
	b.breakTargets = b.breakTargets[:len(b.breakTargets)-1]
	b.loopTargets = b.loopTargets[:len(b.loopTargets)-1]
	operation.post = b.simpleBlock(node.Post)
}

func (b *loweringPlanBuilder) planSwitchStatement(
	operation *plannedOperation,
	node *ast.SwitchStmt,
) {
	operation.kind = planSwitchStatement
	operation.target = b.target()
	operation.init = b.simpleBlock(node.Init)
	if node.Tag != nil {
		tag := b.expression(node.Tag)
		operation.test = b.plannedExpressionBlock(tag)
		operation.expressions = []*plannedExpression{tag}
	}
	b.breakTargets = append(b.breakTargets, operation.target)
	for _, item := range node.Body.List {
		clause := item.(*ast.CaseClause)
		operation.cases = append(operation.cases, b.block(clause.Body))
		for _, expression := range clause.List {
			caseExpression := b.expression(expression)
			operation.entry = append(operation.entry, b.plannedExpressionBlock(caseExpression))
			operation.expressions = append(operation.expressions, caseExpression)
		}
	}
	b.breakTargets = b.breakTargets[:len(b.breakTargets)-1]
}

func (b *loweringPlanBuilder) planTypeSwitchStatement(
	operation *plannedOperation,
	node *ast.TypeSwitchStmt,
) {
	operation.kind = planTypeSwitchStatement
	operation.target = b.target()
	operation.init = b.simpleBlock(node.Init)
	operation.test = b.simpleBlock(node.Assign)
	b.breakTargets = append(b.breakTargets, operation.target)
	for _, item := range node.Body.List {
		operation.cases = append(operation.cases, b.block(item.(*ast.CaseClause).Body))
	}
	b.breakTargets = b.breakTargets[:len(b.breakTargets)-1]
}

func (b *loweringPlanBuilder) planSelectStatement(
	operation *plannedOperation,
	node *ast.SelectStmt,
) {
	operation.kind = planSelectStatement
	operation.target = b.target()
	b.breakTargets = append(b.breakTargets, operation.target)
	for _, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		operation.communications = append(operation.communications, b.communication(clause.Comm))
		operation.entry = append(operation.entry, b.selectEntry(clause.Comm))
		operation.selected = append(operation.selected, b.selectSelected(clause.Comm))
		operation.cases = append(operation.cases, b.block(clause.Body))
	}
	b.breakTargets = b.breakTargets[:len(b.breakTargets)-1]
}

func (b *loweringPlanBuilder) planLabeledStatement(
	operation *plannedOperation,
	node *ast.LabeledStmt,
) {
	operation.kind = planLabeledStatement
	operation.target = b.labels[node.Label.Name]
	operation.body = b.block([]ast.Stmt{node.Stmt})
	operation.labelHasGoto = b.gotos[node.Label.Name]
	if len(operation.body.operations) != 1 {
		return
	}
	child := operation.body.operations[0]
	switch child.kind {
	case planForStatement, planRangeStatement, planSwitchStatement,
		planTypeSwitchStatement, planSelectStatement:
		operation.controlTarget = child.target
		redirectLabeledControl(operation.body, operation.target, child.target)
	}
}

func (b *loweringPlanBuilder) planBranchStatement(
	operation *plannedOperation,
	node *ast.BranchStmt,
) {
	operation.kind = planJump
	switch {
	case node.Label != nil:
		operation.target = b.labels[node.Label.Name]
	case node.Tok == token.CONTINUE:
		operation.target = lastTarget(b.loopTargets)
	case node.Tok == token.BREAK:
		operation.target = lastTarget(b.breakTargets)
	}
}

func (b *loweringPlanBuilder) planDirectStatement(
	operation *plannedOperation,
	statement ast.Stmt,
) {
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
	if update, ok := statement.(*ast.IncDecStmt); ok {
		operation.places = b.assignmentPlaces(update.X)
	}
	b.directStatementBinding(operation)
}

func redirectLabeledControl(block *plannedBlock, source targetID, target targetID) {
	if block == nil {
		return
	}
	for _, operation := range block.operations {
		if operation.kind == planJump && operation.target == source {
			branch, _ := operation.source.(*ast.BranchStmt)
			if branch != nil && (branch.Tok == token.BREAK || branch.Tok == token.CONTINUE) {
				operation.target = target
			}
		}
		for _, child := range []*plannedBlock{
			operation.init, operation.test, operation.body, operation.post,
			operation.otherwise, operation.before, operation.after,
		} {
			redirectLabeledControl(child, source, target)
		}
		for _, child := range operation.cases {
			redirectLabeledControl(child, source, target)
		}
	}
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
		planned := &plannedDeclaration{source: value}
		if len(value.Values) == 1 {
			expected := types.Type(nil)
			if value.Type != nil && len(value.Names) == 1 {
				expected = b.expressionType(value.Type)
			}
			planned.expressions = []*plannedExpression{b.expressionContext(
				value.Values[0], expected, len(value.Names),
			)}
		} else {
			planned.expressions = make([]*plannedExpression, 0, len(value.Values))
			for _, expression := range value.Values {
				expected := types.Type(nil)
				if value.Type != nil {
					expected = b.expressionType(value.Type)
				}
				planned.expressions = append(planned.expressions,
					b.expressionContext(expression, expected, 1))
			}
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
	targets, valid := directBindingTargetCount(operation.source)
	if !valid {
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

func directBindingTargetCount(statement ast.Stmt) (int, bool) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		return len(node.Lhs), node.Tok == token.DEFINE
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR || len(declaration.Specs) != 1 {
			return 0, false
		}
		value, ok := declaration.Specs[0].(*ast.ValueSpec)
		if !ok || value.Type != nil || len(value.Values) != 1 {
			return 0, false
		}
		return len(value.Names), true
	default:
		return 0, false
	}
}

func (b *loweringPlanBuilder) communication(statement ast.Stmt) *plannedCommunication {
	communication := &plannedCommunication{source: statement}
	channel := b.communicationSource(communication, statement)
	if channel == nil {
		return communication
	}
	communication.channel = b.expression(channel)
	channelType := communication.channel.typ
	if len(communication.channel.results) == 1 {
		channelType = communication.channel.results[0].typ
	}
	communication.channelValue = b.newValue(channelType, channel.Pos())
	if channelType == nil {
		return communication
	}
	if named, ok := types.Unalias(channelType).(*types.Named); ok {
		channelType = named.Underlying()
	}
	if channelType, ok := channelType.(*types.Chan); ok {
		if send, ok := statement.(*ast.SendStmt); ok {
			communication.value = b.expressionContext(send.Value, channelType.Elem(), 1)
			communication.sendValue = b.newValue(
				channelType.Elem(), communication.value.source.Pos(),
			)
			communication.sendValue.explicit = communication.value.contextual
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

func (b *loweringPlanBuilder) communicationSource(
	communication *plannedCommunication,
	statement ast.Stmt,
) ast.Expr {
	switch node := statement.(type) {
	case *ast.SendStmt:
		communication.value = b.expression(node.Value)
		return node.Chan
	case *ast.ExprStmt:
		return receiveChannel(node.X)
	case *ast.AssignStmt:
		communication.left = node.Lhs
		communication.token = node.Tok
		communication.targets = b.assignmentPlaces(node.Lhs...)
		if len(node.Rhs) == 1 {
			return receiveChannel(node.Rhs[0])
		}
	}
	return nil
}

func receiveChannel(expression ast.Expr) ast.Expr {
	receive, ok := expression.(*ast.UnaryExpr)
	if ok && receive.Op == token.ARROW {
		return receive.X
	}
	return nil
}

func (b *loweringPlanBuilder) newValue(typ types.Type, position token.Pos) plannedValue {
	value := plannedValue{
		id: valueID(len(b.plan.values) + 1), typ: typ, position: position,
		typeReference: b.typeReference(typ, position),
	}
	b.plan.values = append(b.plan.values, value)
	return value
}

func plannedExpressionProducedType(expression *plannedExpression) types.Type {
	if expression == nil {
		return nil
	}
	if len(expression.results) == 1 {
		return expression.results[0].typ
	}
	if validPlannedType(expression.typ) {
		return expression.typ
	}
	return nil
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
		var value plannedValue
		if expression.work != nil && len(expression.results) == 1 {
			value = expression.results[0]
		} else {
			value = b.newValue(plannedExpressionProducedType(expression), expression.source.Pos())
		}
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
	typ := plannedExpressionProducedType(expression)
	if !validPlannedType(typ) {
		return false
	}
	if _, tuple := typ.(*types.Tuple); tuple {
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
	id := b.nextTarget
	b.plan.targets[id] = plannedTarget{id: id}
	return id
}

func (b *loweringPlanBuilder) indexLabels(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		label, ok := node.(*ast.LabeledStmt)
		if !ok {
			return true
		}
		id := b.target()
		b.labels[label.Label.Name] = id
		b.plan.targets[id] = plannedTarget{
			id: id, label: label.Label.Name, position: label.Colon,
		}
		return true
	})
}

func lastTarget(targets []targetID) targetID {
	if len(targets) == 0 {
		return 0
	}
	return targets[len(targets)-1]
}

func (b *loweringPlanBuilder) sourceScope(position token.Pos) *types.Scope {
	var selected *types.Scope
	for _, scope := range b.unit.info.Scopes {
		if !scope.Contains(position) {
			continue
		}
		if selected == nil || selected.Contains(scope.Pos()) {
			selected = scope
		}
	}
	return selected
}

func (b *loweringPlanBuilder) typeReference(
	typ types.Type,
	position token.Pos,
) plannedTypeReference {
	return capturePlannedTypeReference(
		typ, b.sourceScope(position), position, b.unit.typed, b.unit.info,
		b.source.File, b.function.body,
	)
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
		if len(node.Rhs) == 1 {
			expected := types.Type(nil)
			if len(node.Lhs) == 1 {
				expected = b.expressionType(node.Lhs[0])
			}
			return []*plannedExpression{b.expressionContext(
				node.Rhs[0], expected, len(node.Lhs),
			)}
		}
		if len(node.Lhs) == len(node.Rhs) {
			result := make([]*plannedExpression, 0, len(node.Rhs))
			for index, expression := range node.Rhs {
				result = append(result, b.expressionContext(
					expression, b.expressionType(node.Lhs[index]), 1,
				))
			}
			return result
		}
		return b.expressionList(node.Rhs)
	case *ast.IncDecStmt:
		return b.expressions(node.X)
	case *ast.ExprStmt:
		return b.expressions(node.X)
	case *ast.ReturnStmt:
		return b.returnExpressions(node)
	case *ast.SendStmt:
		return b.sendExpressions(node)
	case *ast.GoStmt:
		return b.expressions(node.Call)
	case *ast.DeferStmt:
		return b.expressions(node.Call)
	case *ast.DeclStmt:
		return b.declarationExpressions(node)
	}
	return nil
}

func (b *loweringPlanBuilder) returnExpressions(node *ast.ReturnStmt) []*plannedExpression {
	result := make([]*plannedExpression, 0, len(node.Results))
	for index, expression := range node.Results {
		expected := b.returnExpectedType(node, index)
		result = append(result, b.expressionContext(
			expression, expected, b.expressionResultCount(expression),
		))
	}
	return result
}

func (b *loweringPlanBuilder) returnExpectedType(node *ast.ReturnStmt, index int) types.Type {
	if len(b.source.FailureReturns[node]) != 0 && b.function.resultType.Len() != 0 {
		return b.function.resultType.At(b.function.resultType.Len() - 1).Type()
	}
	if index < b.function.resultType.Len() {
		return b.function.resultType.At(index).Type()
	}
	return nil
}

func (b *loweringPlanBuilder) sendExpressions(node *ast.SendStmt) []*plannedExpression {
	expected := types.Type(nil)
	channelType := types.Unalias(b.expressionType(node.Chan))
	if named, ok := channelType.(*types.Named); ok {
		channelType = named.Underlying()
	}
	if channel, ok := channelType.(*types.Chan); ok {
		expected = channel.Elem()
	}
	return []*plannedExpression{
		b.expression(node.Chan), b.expressionContext(node.Value, expected, 1),
	}
}

func (b *loweringPlanBuilder) declarationExpressions(node *ast.DeclStmt) []*plannedExpression {
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

func (b *loweringPlanBuilder) expressionList(input []ast.Expr) []*plannedExpression {
	result := make([]*plannedExpression, 0, len(input))
	for _, expression := range input {
		result = append(result, b.expressions(expression)...)
	}
	return result
}

func (b *loweringPlanBuilder) expressionResultCount(expression ast.Expr) int {
	typ := b.expressionType(expression)
	if tuple, ok := typ.(*types.Tuple); ok {
		return tuple.Len()
	}
	if typ == nil {
		return 0
	}
	return 1
}

func (b *loweringPlanBuilder) expressions(expression ast.Expr) []*plannedExpression {
	if expression == nil {
		return nil
	}
	return []*plannedExpression{b.expression(expression)}
}
