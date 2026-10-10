package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *loweringPlanBuilder) expression(expression ast.Expr) *plannedExpression {
	return b.expressionContext(expression, nil, b.expressionResultCount(expression))
}

func (b *loweringPlanBuilder) expressionContext(
	expression ast.Expr,
	expected types.Type,
	resultCount int,
) *plannedExpression {
	result := &plannedExpression{
		kind: planRetainedExpression, source: expression, typ: b.expressionType(expression),
		expected: expected, contextual: b.expressionNeedsContext(expression, expected),
		retainContext: b.expressionRetainsContext(expression),
		resultCount:   resultCount,
	}
	if metadata, function, ok := comprehensionMarker(b.source, expression); ok {
		return b.comprehensionExpression(result, expression, metadata, function)
	}
	if metadata, ok := propagationMarker(b.source, expression); ok {
		return b.propagationExpression(result, expression, metadata)
	}
	if logical, ok := expression.(*ast.BinaryExpr); ok &&
		(logical.Op == token.LAND || logical.Op == token.LOR) {
		return b.logicalExpression(result, logical)
	}
	operands, supported := classifyPlannedExpression(result, expression)
	if !supported {
		return result
	}
	result.operands = b.planOperands(expression, operands, expected)
	result.before = b.orderOperands(result.operands, result.kind == planSliceExpression)
	valueType := result.typ
	if expected != nil && result.contextual {
		valueType = expected
	}
	result.addResult(b, valueType, expression.Pos())
	for len(result.results) < resultCount {
		additional := commaOKResultType(expression, len(result.results))
		if additional == nil {
			break
		}
		result.addResult(b, additional, expression.Pos())
	}
	return result
}

func commaOKResultType(expression ast.Expr, resultIndex int) types.Type {
	if resultIndex != 1 {
		return nil
	}
	switch node := expression.(type) {
	case *ast.IndexExpr, *ast.TypeAssertExpr:
		return types.Typ[types.Bool]
	case *ast.UnaryExpr:
		if node.Op == token.ARROW {
			return types.Typ[types.Bool]
		}
	}
	return nil
}

func (b *loweringPlanBuilder) logicalExpression(
	result *plannedExpression,
	node *ast.BinaryExpr,
) *plannedExpression {
	result.kind = planBinaryExpression
	left := b.expressionContext(node.X, result.typ, 1)
	right := b.expressionContext(node.Y, result.typ, 1)
	result.operands = []*plannedExpression{left, right}
	valueType := result.typ
	if expected := result.expected; expected != nil && result.contextual {
		valueType = expected
	}
	result.addResult(b, valueType, node.Pos())
	if len(result.results) == 0 {
		return result
	}
	output := result.results[0].id
	result.work = &plannedBlock{scope: b.currentScope, operations: []*plannedOperation{
		{kind: planBind, expressions: []*plannedExpression{left}, outputs: []valueID{output}},
		{
			kind: planBranch, inputs: []valueID{output}, operator: node.Op,
			body: &plannedBlock{scope: b.currentScope, operations: []*plannedOperation{{
				kind: planStore, expressions: []*plannedExpression{right},
				inputs: []valueID{output},
			}}},
		},
	}}
	return result
}

func classifyPlannedExpression(result *plannedExpression, expression ast.Expr) ([]ast.Expr, bool) {
	switch node := expression.(type) {
	case *ast.CallExpr:
		result.kind = planCallExpression
		return append([]ast.Expr{node.Fun}, node.Args...), true
	case *ast.BinaryExpr:
		result.kind = planBinaryExpression
		return []ast.Expr{node.X, node.Y}, true
	case *ast.IndexExpr:
		result.kind = planIndexExpression
		return []ast.Expr{node.X, node.Index}, true
	case *ast.IndexListExpr:
		result.kind = planIndexListExpression
		return append([]ast.Expr{node.X}, node.Indices...), true
	case *ast.SelectorExpr:
		result.kind = planSelectorExpression
		return []ast.Expr{node.X}, true
	case *ast.SliceExpr:
		result.kind = planSliceExpression
		return []ast.Expr{node.X, node.Low, node.High, node.Max}, true
	case *ast.ParenExpr:
		result.kind = planParenExpression
		return []ast.Expr{node.X}, true
	case *ast.StarExpr:
		result.kind = planStarExpression
		return []ast.Expr{node.X}, true
	case *ast.UnaryExpr:
		result.kind = planUnaryExpression
		return []ast.Expr{node.X}, true
	case *ast.TypeAssertExpr:
		result.kind = planTypeAssertExpression
		return []ast.Expr{node.X}, true
	case *ast.CompositeLit:
		result.kind = planCompositeExpression
		return node.Elts, true
	case *ast.KeyValueExpr:
		result.kind = planKeyValueExpression
		return []ast.Expr{node.Key, node.Value}, true
	case *ast.FuncLit:
		return nil, false
	default:
		return nil, true
	}
}

func (b *loweringPlanBuilder) planOperands(
	expression ast.Expr,
	operands []ast.Expr,
	expected types.Type,
) []*plannedExpression {
	result := make([]*plannedExpression, 0, len(operands))
	for index, operand := range operands {
		if operand == nil {
			continue
		}
		operandExpected := b.operandExpectedType(expression, index, expected)
		result = append(result, b.expressionContext(
			operand, operandExpected, b.expressionResultCount(operand),
		))
	}
	return result
}

func (b *loweringPlanBuilder) operandExpectedType(
	expression ast.Expr,
	index int,
	expected types.Type,
) types.Type {
	if binary, ok := expression.(*ast.BinaryExpr); ok {
		switch binary.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ,
			token.LAND, token.LOR:
			return nil
		case token.SHL, token.SHR:
			if index == 0 {
				return expected
			}
			return nil
		default:
			return expected
		}
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok || index == 0 {
		return nil
	}
	return b.callOperandExpectedType(call, index-1)
}

func (b *loweringPlanBuilder) callOperandExpectedType(
	call *ast.CallExpr,
	parameter int,
) types.Type {
	signature := b.callSignature(call.Fun)
	if signature == nil || signature.Params().Len() == 0 {
		return nil
	}
	if signature.Variadic() && parameter >= signature.Params().Len()-1 {
		parameter = signature.Params().Len() - 1
		parameterType := signature.Params().At(parameter).Type()
		if slice, ok := parameterType.(*types.Slice); ok && !call.Ellipsis.IsValid() {
			return slice.Elem()
		}
		return parameterType
	}
	if parameter < signature.Params().Len() {
		return signature.Params().At(parameter).Type()
	}
	return nil
}

func (b *loweringPlanBuilder) comprehensionExpression(
	result *plannedExpression,
	expression ast.Expr,
	metadata comprehensionSource,
	function *ast.FuncLit,
) *plannedExpression {
	result.kind = planComprehensionExpression
	result.comprehension = &metadata
	result.function = function
	if function == nil || function.Body == nil {
		result.addResult(b, result.typ, expression.Pos())
		return result
	}
	result.work = b.block(function.Body.List)
	trimComprehensionReturn(result.work)
	b.captureComprehensionResultNames(result, function, metadata.Result)
	b.planComprehensionBuiltins(result, function, metadata)
	_, mapOutput := coreContainerType(result.typ).(*types.Map)
	if metadata.Map && validPlannedType(result.typ) && !mapOutput {
		b.unit.failAt(metadata.Position, "map comprehension needs a map output type")
		return result
	}
	if !metadata.Map && validPlannedType(result.typ) && underlyingSlice(result.typ) == nil {
		b.unit.failAt(metadata.Position, "slice comprehension needs a slice output type")
		return result
	}
	result.addResult(b, result.typ, expression.Pos())
	return result
}

func trimComprehensionReturn(work *plannedBlock) {
	count := len(work.operations)
	if count == 0 {
		return
	}
	if _, ok := work.operations[count-1].source.(*ast.ReturnStmt); ok {
		work.operations = work.operations[:count-1]
	}
}

func (b *loweringPlanBuilder) captureComprehensionResultNames(
	result *plannedExpression,
	function *ast.FuncLit,
	name string,
) {
	var resultObject types.Object
	ast.Inspect(function.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Name != name {
			return true
		}
		object := b.unit.info.ObjectOf(identifier)
		if resultObject == nil && b.unit.info.Defs[identifier] != nil {
			resultObject = object
		}
		if resultObject == nil || object == resultObject {
			result.resultNames = append(result.resultNames, identifier)
		}
		return true
	})
}

func (b *loweringPlanBuilder) planComprehensionBuiltins(
	result *plannedExpression,
	function *ast.FuncLit,
	metadata comprehensionSource,
) {
	makeCall, outer, valid := b.comprehensionParts(function.Body)
	if !valid {
		return
	}
	if !validPlannedType(result.typ) && len(makeCall.Args) != 0 {
		result.typ = b.expressionType(makeCall.Args[0])
	}
	result.builtins = append(result.builtins, &plannedBuiltin{
		call: makeCall, name: "make", position: metadata.Position,
	})
	assignment, appendCall := directComprehensionAppend(outer, metadata.Map)
	if appendCall == nil {
		return
	}
	result.builtins = append(result.builtins, &plannedBuiltin{
		call: appendCall, name: "append", position: metadata.Position,
	})
	b.planExactComprehension(result, function, makeCall, outer, assignment, appendCall, metadata)
}

func directComprehensionAppend(
	outer *ast.RangeStmt,
	isMap bool,
) (*ast.AssignStmt, *ast.CallExpr) {
	terminal := comprehensionTerminal(outer.Body)
	if isMap || terminal != outer.Body || len(terminal.List) != 1 {
		return nil, nil
	}
	assignment, ok := terminal.List[0].(*ast.AssignStmt)
	if !ok || len(assignment.Rhs) != 1 {
		return nil, nil
	}
	appendCall, _ := assignment.Rhs[0].(*ast.CallExpr)
	return assignment, appendCall
}

func (b *loweringPlanBuilder) planExactComprehension(
	result *plannedExpression,
	function *ast.FuncLit,
	makeCall *ast.CallExpr,
	outer *ast.RangeStmt,
	assignment *ast.AssignStmt,
	appendCall *ast.CallExpr,
	metadata comprehensionSource,
) {
	if len(makeCall.Args) != 2 || len(function.Body.List) != 3 || len(outer.Body.List) != 1 ||
		underlyingSlice(b.expressionType(outer.X)) == nil ||
		underlyingSlice(b.expressionType(makeCall.Args[0])) == nil {
		return
	}
	for _, operation := range result.work.operations {
		if operation.source != outer || len(operation.expressions) != 1 {
			continue
		}
		sourcePlan := operation.expressions[0]
		sourceValue := b.newValue(plannedExpressionProducedType(sourcePlan), outer.X.Pos())
		sourcePlan.materialized = sourceValue.id
		result.work.operations = append([]*plannedOperation{{
			kind: planEvaluate, expressions: []*plannedExpression{sourcePlan},
			outputs: []valueID{sourceValue.id}, preferred: "source",
		}}, result.work.operations...)
		result.exact = &plannedExactComprehension{
			makeCall: makeCall, outer: outer, assignment: assignment,
			appendCall: appendCall, source: sourceValue, position: metadata.Position,
			identity: sameExpressionObject(b.unit.info, outer.Value, appendCall.Args[1]),
		}
		if result.exact.identity {
			operation.kind = planCopy
			operation.inputs = []valueID{sourceValue.id}
			operation.copyTarget = assignment.Lhs[0]
		}
		return
	}
}

func (b *loweringPlanBuilder) propagationExpression(
	result *plannedExpression,
	expression ast.Expr,
	metadata propagationSource,
) *plannedExpression {
	result.kind = planPropagationExpression
	result.propagation = &metadata
	result.work = &plannedBlock{scope: b.currentScope}
	marker := expression.(*ast.CallExpr)
	call, valid := unwrappedCompilerCall(marker.Args[0])
	if !valid {
		return result
	}
	callPlan := b.expression(call)
	result.operands = append(result.operands, callPlan)
	signature := b.callSignature(call.Fun)
	if signature == nil {
		return result
	}
	for index := 0; index < signature.Results().Len(); index++ {
		result.addResult(b, signature.Results().At(index).Type(), expression.Pos())
	}
	if len(result.results) == 0 {
		return result
	}
	b.planPropagationWork(result, callPlan, metadata)
	return result
}

func (b *loweringPlanBuilder) planPropagationWork(
	result *plannedExpression,
	callPlan *plannedExpression,
	metadata propagationSource,
) {
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

func sameExpressionObject(info *types.Info, left ast.Expr, right ast.Expr) bool {
	leftIdentifier, leftOK := left.(*ast.Ident)
	rightIdentifier, rightOK := right.(*ast.Ident)
	return leftOK && rightOK && info.ObjectOf(leftIdentifier) != nil &&
		info.ObjectOf(leftIdentifier) == info.ObjectOf(rightIdentifier)
}

func (b *loweringPlanBuilder) expressionRetainsContext(expression ast.Expr) bool {
	if identifier, ok := expression.(*ast.Ident); ok && identifier.Name == "nil" {
		return true
	}
	value, ok := b.unit.info.Types[expression]
	return ok && value.Value != nil
}

func (b *loweringPlanBuilder) expressionNeedsContext(
	expression ast.Expr,
	expected types.Type,
) bool {
	if expected == nil {
		return false
	}
	switch node := expression.(type) {
	case *ast.BasicLit:
		return true
	case *ast.BinaryExpr:
		switch node.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ,
			token.LAND, token.LOR:
			return isBooleanType(expected)
		case token.SHL, token.SHR:
			return b.expressionNeedsContext(node.X, expected)
		default:
			return isUntypedType(b.expressionType(node))
		}
	case *ast.Ident:
		return node.Name == "nil"
	case *ast.UnaryExpr:
		return isUntypedType(b.expressionType(node))
	case *ast.ParenExpr:
		return b.expressionNeedsContext(node.X, expected)
	case *ast.CompositeLit:
		return node.Type == nil
	}
	return false
}

func isBooleanType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if named, ok := types.Unalias(typ).(*types.Named); ok {
		typ = named.Underlying()
	}
	basic, ok := types.Unalias(typ).(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func isUntypedType(typ types.Type) bool {
	basic, ok := types.Unalias(typ).(*types.Basic)
	return ok && basic.Info()&types.IsUntyped != 0
}

func (b *loweringPlanBuilder) orderOperands(
	operands []*plannedExpression,
	addressArray bool,
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
		operation := b.operandOrderOperation(operand, index, laterWork, addressArray)
		if operation != nil {
			block.operations = append(block.operations, operation)
		}
	}
	if len(block.operations) == 0 {
		return nil
	}
	return block
}

func (b *loweringPlanBuilder) operandOrderOperation(
	operand *plannedExpression,
	index int,
	laterWork bool,
	addressArray bool,
) *plannedOperation {
	if laterWork {
		if pointer := arrayPointerOperand(operand); pointer != nil && len(pointer.results) == 1 {
			pointer.materialized = pointer.results[0].id
			return &plannedOperation{
				kind: planEvaluate, expressions: []*plannedExpression{pointer},
				outputs: []valueID{pointer.results[0].id},
			}
		}
	}
	if addressArray && index == 0 && laterWork && underlyingArray(operand.typ) != nil {
		place := b.assignmentPlace(operand.source)
		operand.place = place
		prepare := &plannedBlock{scope: b.currentScope}
		b.planPlacePreparation(place, prepare)
		return &plannedOperation{
			kind: planPlaceReady, places: []*plannedPlace{place}, before: prepare,
		}
	}
	if len(operand.results) != 1 {
		return nil
	}
	if laterWork {
		b.planBooleanContextAdapter(operand)
	}
	operand.materialized = operand.results[0].id
	return &plannedOperation{
		kind: planEvaluate, expressions: []*plannedExpression{operand},
		outputs: []valueID{operand.results[0].id},
	}
}

func (b *loweringPlanBuilder) planBooleanContextAdapter(expression *plannedExpression) {
	if !b.needsBooleanContextAdapter(expression.expected) {
		return
	}
	if !expression.contextual && (expression.kind == planCallExpression ||
		expression.kind == planPropagationExpression) {
		return
	}
	b.normalizeBooleanStorage(expression)
}

func (b *loweringPlanBuilder) normalizeBooleanStorage(expression *plannedExpression) {
	expression.booleanAdapter = true
	if expression.kind != planPropagationExpression {
		for _, operand := range expression.operands {
			if b.expressionNeedsBooleanAdapter(operand) {
				b.normalizeBooleanStorage(operand)
			}
		}
	}
	if len(expression.results) != 1 {
		return
	}
	if expression.kind == planPropagationExpression {
		b.convertPropagationBooleanResult(expression)
		return
	}
	value := &b.plan.values[expression.results[0].id-1]
	value.typ = types.Typ[types.Bool]
	value.typeReference = b.typeReference(value.typ, value.position)
	expression.results[0] = *value
}

func (b *loweringPlanBuilder) convertPropagationBooleanResult(
	expression *plannedExpression,
) {
	source := expression.results[0]
	converted := b.newValue(types.Typ[types.Bool], source.position)
	expression.work.operations = append(expression.work.operations, &plannedOperation{
		kind: planBooleanConvert, inputs: []valueID{source.id},
		outputs: []valueID{converted.id},
	})
	expression.results[0] = converted
}

func (b *loweringPlanBuilder) expressionNeedsBooleanAdapter(
	expression *plannedExpression,
) bool {
	if b.needsBooleanContextAdapter(expression.expected) ||
		b.needsBooleanContextAdapter(expression.typ) {
		return true
	}
	for _, result := range expression.results {
		if b.needsBooleanContextAdapter(result.typ) {
			return true
		}
	}
	return false
}

func (b *loweringPlanBuilder) needsBooleanContextAdapter(typ types.Type) bool {
	if typ == nil {
		return false
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return false
	}
	basic, ok := named.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func arrayPointerOperand(expression *plannedExpression) *plannedExpression {
	if expression == nil || underlyingArray(expression.typ) == nil {
		return nil
	}
	switch expression.source.(type) {
	case *ast.StarExpr:
		if len(expression.operands) == 1 {
			return expression.operands[0]
		}
	case *ast.ParenExpr:
		if len(expression.operands) == 1 {
			return arrayPointerOperand(expression.operands[0])
		}
	}
	return nil
}

func underlyingArray(typ types.Type) *types.Array {
	array, _ := coreContainerType(typ).(*types.Array)
	return array
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
	value := plannedValue{
		id: valueID(len(b.plan.values) + 1), typ: typ, position: position,
		typeReference: b.typeReference(typ, position),
	}
	b.plan.values = append(b.plan.values, value)
	e.results = append(e.results, value)
}
