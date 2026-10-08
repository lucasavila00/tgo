package tgolint

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

func (c *checker) reportResult(position token.Pos, format string, arguments ...any) {
	message := fmt.Sprintf(format, arguments...)
	key := diagnosticKey{position: position, message: message}
	if c.reported[key] {
		return
	}
	c.reported[key] = true
	c.pass.Reportf(position, "%s", message)
}

type checkedResult struct {
	failure    types.Object
	model      *modelFact
	safe       bool
	validProof bool
	presence   bool
}

type checkedState map[types.Object]checkedResult

func (c *checker) checkConstructors(body *ast.BlockStmt) {
	previous := c.escaped
	c.escaped = escapedObjects(c.pass.TypesInfo, body)
	defer func() {
		c.escaped = previous
	}()
	c.checkedBlock(body.List, make(checkedState))
}

// escapedObjects finds local variables that an address or nested function can use.
// A result pair cannot use these variables as a stable proof.
func escapedObjects(info *types.Info, body *ast.BlockStmt) map[types.Object]token.Pos {
	escaped := make(map[types.Object]token.Pos)
	record := func(object types.Object, position token.Pos) {
		if object == nil {
			return
		}
		first, found := escaped[object]
		if !found || position < first {
			escaped[object] = position
		}
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				record(info.ObjectOf(identifier(node.X)), node.Pos())
			}
		case *ast.FuncLit:
			ast.Inspect(node.Body, func(captured ast.Node) bool {
				name, ok := captured.(*ast.Ident)
				if ok {
					record(info.Uses[name], node.Pos())
				}
				return true
			})
			return false
		}
		return true
	})
	return escaped
}

func (c *checker) checkedBlock(statements []ast.Stmt, state checkedState) bool {
	for _, statement := range statements {
		if c.checkedStatement(statement, state) {
			return true
		}
		if c.statementsTerminate([]ast.Stmt{statement}) {
			return true
		}
	}
	return false
}

func (c *checker) checkedStatement(statement ast.Stmt, state checkedState) bool {
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		if c.checkedAssignment(statement, state) {
			return false
		}
		c.checkResultUses(statement.Rhs, state, nil)
		for _, expression := range statement.Lhs {
			c.invalidateEscapedProofs(expression, state)
		}
		c.invalidateAssignments(statement.Lhs, state)
	case *ast.DeclStmt:
		c.checkedDeclaration(statement, state)
	case *ast.ReturnStmt:
		c.checkedReturn(statement, state)
		return true
	case *ast.IfStmt:
		return c.checkedIf(statement, state)
	case *ast.BlockStmt:
		return c.checkedBlock(statement.List, state)
	default:
		return c.checkedControlStatement(statement, state)
	}
	return false
}

func (c *checker) checkedControlStatement(statement ast.Stmt, state checkedState) bool {
	switch statement := statement.(type) {
	case *ast.ForStmt:
		if statement.Init != nil {
			c.checkedStatement(statement.Init, state)
		}
		c.checkedLoop(statement, state)
	case *ast.RangeStmt:
		c.checkedRange(statement, state)
	case *ast.SwitchStmt:
		return c.checkedSwitch(statement, state)
	case *ast.TypeSwitchStmt:
		return c.checkedTypeSwitch(statement, state)
	case *ast.SelectStmt:
		var exits []checkedState
		for _, item := range statement.Body.List {
			clause := item.(*ast.CommClause)
			branch := cloneCheckedState(state)
			if clause.Comm != nil {
				c.checkedStatement(clause.Comm, branch)
			}
			if !c.checkedBlock(clause.Body, branch) {
				exits = append(exits, branch)
			}
		}
		return replaceWithJoinedStates(state, exits)
	case *ast.LabeledStmt:
		return c.checkedStatement(statement.Stmt, state)
	case *ast.BranchStmt, *ast.EmptyStmt:
		return false
	default:
		c.checkResultUses([]ast.Expr{statementExpression(statement)}, state, nil)
	}
	return false
}

func (c *checker) checkedLoop(statement *ast.ForStmt, state checkedState) {
	entry := cloneCheckedState(state)
	loop := cloneCheckedState(entry)
	for {
		c.checkResultUses([]ast.Expr{statement.Cond}, loop, nil)
		iteration := cloneCheckedState(loop)
		trueProofs, falseProofs := c.resultProofs(statement.Cond, loop)
		for failure := range trueProofs {
			proveResult(iteration, failure)
		}
		if !c.checkedBlock(statement.Body.List, iteration) && statement.Post != nil {
			c.checkedStatement(statement.Post, iteration)
		}
		next := joinCheckedStates(entry, iteration)
		if equalCheckedStates(loop, next) {
			for failure := range falseProofs {
				proveResult(next, failure)
			}
			replaceCheckedState(state, next)
			return
		}
		loop = next
	}
}

func (c *checker) checkedRange(statement *ast.RangeStmt, state checkedState) {
	c.checkResultUses([]ast.Expr{statement.X}, state, nil)
	entry := cloneCheckedState(state)
	loop := cloneCheckedState(entry)
	for {
		iteration := cloneCheckedState(loop)
		c.invalidateAssignments([]ast.Expr{statement.Key, statement.Value}, iteration)
		c.checkedBlock(statement.Body.List, iteration)
		next := joinCheckedStates(entry, iteration)
		if equalCheckedStates(loop, next) {
			replaceCheckedState(state, next)
			return
		}
		loop = next
	}
}

func statementExpression(statement ast.Stmt) ast.Expr {
	switch statement := statement.(type) {
	case *ast.ExprStmt:
		return statement.X
	case *ast.GoStmt:
		return statement.Call
	case *ast.DeferStmt:
		return statement.Call
	case *ast.SendStmt:
		return &ast.CompositeLit{Elts: []ast.Expr{statement.Chan, statement.Value}}
	case *ast.IncDecStmt:
		return statement.X
	}
	return nil
}

func (c *checker) checkedAssignment(assignment *ast.AssignStmt, state checkedState) bool {
	if len(assignment.Rhs) != 1 {
		return false
	}
	expression := assignment.Rhs[0]
	call, callOK := expression.(*ast.CallExpr)
	if callOK {
		model := c.checkedCall(call)
		if model != nil {
			c.checked[call] = true
			c.checkResultUses(call.Args, state, nil)
			if len(assignment.Lhs) != 2 {
				c.reportCheckedCall(call)
				return true
			}
			c.bindCheckedResults(assignment.Lhs[0], assignment.Lhs[1], model, state)
			return true
		}
	}
	presenceModel := c.presenceModel(expression)
	if presenceModel == nil {
		return false
	}
	c.presence[expression] = true
	c.checkResultUses([]ast.Expr{expression}, state, nil)
	if len(assignment.Lhs) != 2 {
		return true
	}
	c.bindPresenceResults(assignment.Lhs[0], assignment.Lhs[1], presenceModel, state)
	return true
}

func (c *checker) checkedDeclaration(statement *ast.DeclStmt, state checkedState) {
	declaration, ok := statement.Decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, item := range declaration.Specs {
		specification, ok := item.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if len(specification.Values) != 1 || len(specification.Names) != 2 {
			c.checkResultUses(specification.Values, state, nil)
			continue
		}
		call, ok := specification.Values[0].(*ast.CallExpr)
		if ok {
			model := c.checkedCall(call)
			if model != nil {
				c.checked[call] = true
				c.checkResultUses(call.Args, state, nil)
				c.bindCheckedResults(
					specification.Names[0],
					specification.Names[1],
					model,
					state,
				)
				continue
			}
		}
		expression := specification.Values[0]
		model := c.presenceModel(expression)
		if model != nil {
			c.presence[expression] = true
			c.checkResultUses([]ast.Expr{expression}, state, nil)
			c.bindPresenceResults(
				specification.Names[0],
				specification.Names[1],
				model,
				state,
			)
			continue
		}
		c.checkResultUses(specification.Values, state, nil)
	}
}

func (c *checker) bindCheckedResults(
	valueExpression ast.Expr,
	errorExpression ast.Expr,
	model *modelFact,
	state checkedState,
) {
	c.invalidateAssignments([]ast.Expr{valueExpression, errorExpression}, state)
	errorName, errorOK := errorExpression.(*ast.Ident)
	if !errorOK || errorName.Name == "_" {
		c.reportResult(errorExpression.Pos(),
			"error for tgo %s %s result must not be discarded",
			model.Kind, model.Name)
		return
	}
	errorObject := c.pass.TypesInfo.ObjectOf(errorName)
	valueName, valueOK := valueExpression.(*ast.Ident)
	if !valueOK {
		c.reportResult(valueExpression.Pos(),
			"tgo %s %s result must first use a local variable",
			model.Kind, model.Name)
		return
	}
	if valueName.Name == "_" {
		return
	}
	valueObject := c.pass.TypesInfo.ObjectOf(valueName)
	if valueObject == nil || errorObject == nil {
		return
	}
	if escapedBefore(c.escaped, valueObject, valueExpression.Pos()) ||
		escapedBefore(c.escaped, errorObject, errorExpression.Pos()) {
		c.reportResult(valueExpression.Pos(),
			"tgo %s %s result variables must not have aliases",
			model.Kind, model.Name)
	}
	state[valueObject] = checkedResult{
		failure:    errorObject,
		model:      model,
		validProof: true,
	}
}

func (c *checker) presenceModel(expression ast.Expr) *modelFact {
	var valueType types.Type
	if tuple, ok := c.pass.TypesInfo.TypeOf(expression).(*types.Tuple); ok &&
		tuple.Len() == 2 && isBoolean(tuple.At(1).Type()) {
		valueType = tuple.At(0).Type()
	}
	if valueType != nil {
		model, invalid := c.zeroInvalid(valueType)
		if invalid {
			return model
		}
		return nil
	}
	switch expression := expression.(type) {
	case *ast.IndexExpr:
		mapping, ok := coreType(c.pass.TypesInfo.TypeOf(expression.X)).(*types.Map)
		if !ok {
			return nil
		}
		valueType = mapping.Elem()
	case *ast.UnaryExpr:
		if expression.Op != token.ARROW {
			return nil
		}
		valueType = firstType(c.pass.TypesInfo.TypeOf(expression))
	case *ast.TypeAssertExpr:
		if expression.Type == nil {
			return nil
		}
		valueType = firstType(c.pass.TypesInfo.TypeOf(expression))
	default:
		return nil
	}
	model, invalid := c.zeroInvalid(valueType)
	if !invalid {
		return nil
	}
	return model
}

func isBoolean(typ types.Type) bool {
	basic, ok := types.Unalias(typ).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func (c *checker) bindPresenceResults(
	valueExpression ast.Expr,
	presenceExpression ast.Expr,
	model *modelFact,
	state checkedState,
) {
	c.invalidateAssignments([]ast.Expr{valueExpression, presenceExpression}, state)
	presenceName, presenceOK := presenceExpression.(*ast.Ident)
	if !presenceOK || presenceName.Name == "_" {
		c.reportResult(presenceExpression.Pos(),
			"ok result for tgo %s %s presence read must not be discarded",
			model.Kind, model.Name)
		return
	}
	valueName, valueOK := valueExpression.(*ast.Ident)
	if !valueOK {
		c.reportResult(valueExpression.Pos(),
			"tgo %s %s presence value must first use a local variable",
			model.Kind, model.Name)
		return
	}
	if valueName.Name == "_" {
		return
	}
	valueObject := c.pass.TypesInfo.ObjectOf(valueName)
	presenceObject := c.pass.TypesInfo.ObjectOf(presenceName)
	if valueObject == nil || presenceObject == nil {
		return
	}
	if escapedBefore(c.escaped, valueObject, valueExpression.Pos()) ||
		escapedBefore(c.escaped, presenceObject, presenceExpression.Pos()) {
		c.reportResult(valueExpression.Pos(),
			"tgo %s %s presence variables must not have aliases",
			model.Kind, model.Name)
	}
	state[valueObject] = checkedResult{
		failure:    presenceObject,
		model:      model,
		validProof: true,
		presence:   true,
	}
}

func escapedBefore(
	escaped map[types.Object]token.Pos,
	object types.Object,
	position token.Pos,
) bool {
	escape, found := escaped[object]
	return found && escape < position
}

func (c *checker) checkedReturn(statement *ast.ReturnStmt, state checkedState) {
	if len(statement.Results) == 1 {
		if call, ok := statement.Results[0].(*ast.CallExpr); ok &&
			c.checkedCall(call) != nil {
			c.checked[call] = true
			c.checkResultUses(call.Args, state, nil)
			return
		}
	}
	skip := make(map[*ast.Ident]bool)
	for index := 0; index+1 < len(statement.Results); index++ {
		value, valueOK := statement.Results[index].(*ast.Ident)
		failure, errorOK := statement.Results[index+1].(*ast.Ident)
		if !valueOK || !errorOK {
			continue
		}
		result, ok := state[c.pass.TypesInfo.ObjectOf(value)]
		if ok && result.failure == c.pass.TypesInfo.ObjectOf(failure) && result.validProof {
			skip[value] = true
		}
	}
	c.checkResultUses(statement.Results, state, skip)
}

func (c *checker) checkedIf(statement *ast.IfStmt, state checkedState) bool {
	if statement.Init != nil {
		c.checkedStatement(statement.Init, state)
	}
	c.checkResultUses([]ast.Expr{statement.Cond}, state, nil)
	trueState := cloneCheckedState(state)
	falseState := cloneCheckedState(state)
	trueProofs, falseProofs := c.resultProofs(statement.Cond, state)
	for failure := range trueProofs {
		proveResult(trueState, failure)
	}
	for failure := range falseProofs {
		proveResult(falseState, failure)
	}
	trueStops := c.checkedBlock(statement.Body.List, trueState) ||
		c.statementsTerminate(statement.Body.List)
	falseStops := false
	if statement.Else != nil {
		falseStops = c.checkedStatement(statement.Else, falseState)
		if !falseStops {
			falseStops = c.statementsTerminate([]ast.Stmt{statement.Else})
		}
	}
	switch {
	case trueStops && falseStops:
		return true
	case trueStops:
		replaceCheckedState(state, falseState)
	case falseStops:
		replaceCheckedState(state, trueState)
	default:
		mergeCheckedStates(state, trueState, falseState)
	}
	return false
}

type proofSet map[types.Object]bool

// resultProofs returns proofs that hold when an expression is true or false.
// It follows short-circuit Boolean control flow.
func (c *checker) resultProofs(
	expression ast.Expr,
	state checkedState,
) (proofSet, proofSet) {
	if parentheses, ok := expression.(*ast.ParenExpr); ok {
		return c.resultProofs(parentheses.X, state)
	}
	if negation, ok := expression.(*ast.UnaryExpr); ok && negation.Op == token.NOT {
		trueProofs, falseProofs := c.resultProofs(negation.X, state)
		return falseProofs, trueProofs
	}
	if binary, ok := expression.(*ast.BinaryExpr); ok {
		switch binary.Op {
		case token.LAND:
			leftTrue, leftFalse := c.resultProofs(binary.X, state)
			rightTrue, rightFalse := c.resultProofs(binary.Y, state)
			return unionProofs(leftTrue, rightTrue), intersectProofs(
				leftFalse,
				unionProofs(leftTrue, rightFalse),
			)
		case token.LOR:
			leftTrue, leftFalse := c.resultProofs(binary.X, state)
			rightTrue, rightFalse := c.resultProofs(binary.Y, state)
			return intersectProofs(
				leftTrue,
				unionProofs(leftFalse, rightTrue),
			), unionProofs(leftFalse, rightFalse)
		}
	}
	failure, successOnTrue := c.atomicResultProof(expression, state)
	if failure == nil {
		return nil, nil
	}
	if successOnTrue {
		return proofSet{failure: true}, nil
	}
	return nil, proofSet{failure: true}
}

func (c *checker) atomicResultProof(
	expression ast.Expr,
	state checkedState,
) (types.Object, bool) {
	if name, ok := expression.(*ast.Ident); ok {
		object := c.pass.TypesInfo.ObjectOf(name)
		if c.hasValidPresenceProof(state, object) {
			return object, true
		}
		return nil, false
	}
	binary, ok := expression.(*ast.BinaryExpr)
	if !ok || binary.Op != token.EQL && binary.Op != token.NEQ {
		return nil, false
	}
	if failure, success := c.binaryErrorProof(binary, state); failure != nil {
		return failure, success
	}
	return c.binaryPresenceProof(binary, state)
}

func (c *checker) binaryErrorProof(
	binary *ast.BinaryExpr,
	state checkedState,
) (types.Object, bool) {
	name, nilName := errorAndNil(binary.X, binary.Y)
	if name == nil {
		name, nilName = errorAndNil(binary.Y, binary.X)
	}
	if name == nil || nilName == nil ||
		c.pass.TypesInfo.Uses[nilName] != types.Universe.Lookup("nil") {
		return nil, false
	}
	object := c.pass.TypesInfo.ObjectOf(name)
	for _, result := range state {
		if !result.presence && result.failure == object && result.validProof {
			return object, binary.Op == token.EQL
		}
	}
	return nil, false
}

func (c *checker) binaryPresenceProof(
	binary *ast.BinaryExpr,
	state checkedState,
) (types.Object, bool) {
	name, value, ok := booleanComparison(c.pass.TypesInfo, binary.X, binary.Y)
	if !ok {
		name, value, ok = booleanComparison(c.pass.TypesInfo, binary.Y, binary.X)
	}
	if !ok {
		return nil, false
	}
	object := c.pass.TypesInfo.ObjectOf(name)
	if !c.hasValidPresenceProof(state, object) {
		return nil, false
	}
	successOnTrue := value
	if binary.Op == token.NEQ {
		successOnTrue = !successOnTrue
	}
	return object, successOnTrue
}

func booleanComparison(
	info *types.Info,
	nameExpression ast.Expr,
	valueExpression ast.Expr,
) (*ast.Ident, bool, bool) {
	name, ok := nameExpression.(*ast.Ident)
	if !ok {
		return nil, false, false
	}
	value := info.Types[valueExpression].Value
	if value == nil || value.Kind() != constant.Bool {
		return nil, false, false
	}
	return name, constant.BoolVal(value), true
}

func unionProofs(left, right proofSet) proofSet {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	result := make(proofSet, len(left)+len(right))
	for proof := range left {
		result[proof] = true
	}
	for proof := range right {
		result[proof] = true
	}
	return result
}

func intersectProofs(left, right proofSet) proofSet {
	var result proofSet
	for proof := range left {
		if right[proof] {
			if result == nil {
				result = make(proofSet)
			}
			result[proof] = true
		}
	}
	return result
}

func (c *checker) hasValidPresenceProof(state checkedState, object types.Object) bool {
	for _, result := range state {
		if result.presence && result.failure == object && result.validProof {
			return true
		}
	}
	return false
}

func errorAndNil(errorExpression, nilExpression ast.Expr) (*ast.Ident, *ast.Ident) {
	name, ok := errorExpression.(*ast.Ident)
	if !ok {
		return nil, nil
	}
	nilName, ok := nilExpression.(*ast.Ident)
	if !ok || nilName.Name != "nil" {
		return nil, nil
	}
	return name, nilName
}

func proveResult(state checkedState, failure types.Object) {
	for object, result := range state {
		if result.failure == failure && result.validProof {
			result.safe = true
			state[object] = result
		}
	}
}

func (c *checker) checkResultUses(
	expressions []ast.Expr,
	state checkedState,
	skip map[*ast.Ident]bool,
) {
	for _, expression := range expressions {
		if expression == nil {
			continue
		}
		c.invalidateEscapedProofs(expression, state)
		ast.Inspect(expression, func(node ast.Node) bool {
			name, ok := node.(*ast.Ident)
			if !ok || skip[name] {
				return true
			}
			result, found := state[c.pass.TypesInfo.Uses[name]]
			if found && !result.safe {
				if result.presence {
					c.reportResult(name.Pos(),
						"tgo %s %s presence value is used before ok is proved true",
						result.model.Kind, result.model.Name)
				} else {
					c.reportResult(name.Pos(),
						"tgo %s %s value is used before its error is proved nil",
						result.model.Kind, result.model.Name)
				}
			}
			return true
		})
	}
}

// invalidateEscapedProofs rejects a later proof through a writable alias.
// An address or closure can change an error or ok value before the test.
func (c *checker) invalidateEscapedProofs(expression ast.Expr, state checkedState) {
	ast.Inspect(expression, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				c.invalidateProof(c.pass.TypesInfo.ObjectOf(identifier(node.X)), state)
			}
		case *ast.FuncLit:
			for _, result := range state {
				if capturesObject(c.pass.TypesInfo, node.Body, result.failure) {
					c.invalidateProof(result.failure, state)
				}
			}
			return false
		}
		return true
	})
}

func identifier(expression ast.Expr) *ast.Ident {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression
	case *ast.ParenExpr:
		return identifier(expression.X)
	default:
		return nil
	}
}

func (c *checker) invalidateProof(object types.Object, state checkedState) {
	if object == nil {
		return
	}
	for value, result := range state {
		if result.failure == object && !result.safe {
			result.validProof = false
			state[value] = result
		}
	}
}

func (c *checker) invalidateAssignments(expressions []ast.Expr, state checkedState) {
	for _, expression := range expressions {
		name, ok := expression.(*ast.Ident)
		if !ok || name.Name == "_" {
			continue
		}
		object := c.pass.TypesInfo.ObjectOf(name)
		delete(state, object)
		c.invalidateProof(object, state)
	}
}

func (c *checker) checkedSwitch(statement *ast.SwitchStmt, state checkedState) bool {
	if statement.Init != nil {
		c.checkedStatement(statement.Init, state)
	}
	c.checkResultUses([]ast.Expr{statement.Tag}, state, nil)
	entry := cloneCheckedState(state)
	var exits []checkedState
	var carried checkedState
	hasDefault := false
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		branch := cloneCheckedState(entry)
		if carried != nil {
			branch = joinCheckedStates(branch, carried)
		}
		c.checkResultUses(clause.List, branch, nil)
		stops := c.checkedBlock(clause.Body, branch)
		if len(clause.List) == 0 {
			hasDefault = true
		}
		if clauseFallthrough(clause) != nil {
			carried = branch
			continue
		}
		carried = nil
		if !stops {
			exits = append(exits, branch)
		}
	}
	if !hasDefault {
		exits = append(exits, entry)
	}
	return replaceWithJoinedStates(state, exits)
}

func (c *checker) checkedTypeSwitch(statement *ast.TypeSwitchStmt, state checkedState) bool {
	if statement.Init != nil {
		c.checkedStatement(statement.Init, state)
	}
	if statement.Assign != nil {
		c.checkedStatement(statement.Assign, state)
	}
	entry := cloneCheckedState(state)
	var exits []checkedState
	hasDefault := false
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		branch := cloneCheckedState(entry)
		if !c.checkedBlock(clause.Body, branch) {
			exits = append(exits, branch)
		}
		if len(clause.List) == 0 {
			hasDefault = true
		}
	}
	if !hasDefault {
		exits = append(exits, entry)
	}
	return replaceWithJoinedStates(state, exits)
}

func cloneCheckedState(state checkedState) checkedState {
	clone := make(checkedState, len(state))
	for object, result := range state {
		clone[object] = result
	}
	return clone
}

func replaceCheckedState(target, source checkedState) {
	clear(target)
	for object, result := range source {
		target[object] = result
	}
}

func mergeCheckedStates(target, left, right checkedState) {
	clear(target)
	for object, leftResult := range left {
		rightResult, ok := right[object]
		if !ok {
			if !leftResult.safe {
				leftResult.validProof = false
				target[object] = leftResult
			}
			continue
		}
		if leftResult.failure != rightResult.failure {
			if !leftResult.safe || !rightResult.safe {
				leftResult.safe = false
				leftResult.validProof = false
				target[object] = leftResult
			}
			continue
		}
		leftResult.safe = leftResult.safe && rightResult.safe
		leftResult.validProof = leftResult.validProof && rightResult.validProof
		target[object] = leftResult
	}
	for object, rightResult := range right {
		if _, ok := left[object]; !ok && !rightResult.safe {
			rightResult.validProof = false
			target[object] = rightResult
		}
	}
}

func joinCheckedStates(left, right checkedState) checkedState {
	joined := make(checkedState)
	mergeCheckedStates(joined, left, right)
	return joined
}

func replaceWithJoinedStates(target checkedState, states []checkedState) bool {
	if len(states) == 0 {
		clear(target)
		return true
	}
	joined := cloneCheckedState(states[0])
	for _, state := range states[1:] {
		joined = joinCheckedStates(joined, state)
	}
	replaceCheckedState(target, joined)
	return false
}

func equalCheckedStates(left, right checkedState) bool {
	if len(left) != len(right) {
		return false
	}
	for object, result := range left {
		if right[object] != result {
			return false
		}
	}
	return true
}

// checkedCall finds a call that returns a zero-invalid tgo value and an error.
// It includes constructors, decoders, wrappers, and function values.
func (c *checker) checkedCall(call *ast.CallExpr) *modelFact {
	tuple, ok := c.pass.TypesInfo.TypeOf(call).(*types.Tuple)
	if !ok || tuple.Len() != 2 {
		return nil
	}
	model, invalid := c.zeroInvalid(tuple.At(0).Type())
	if !invalid {
		return nil
	}
	if !types.Identical(tuple.At(1).Type(), types.Universe.Lookup("error").Type()) {
		return nil
	}
	return model
}

func (c *checker) reportCheckedCall(call *ast.CallExpr) {
	model := c.checkedCall(call)
	if model == nil {
		return
	}
	c.reportResult(call.Pos(),
		"tgo %s %s result error must be checked or returned",
		model.Kind, model.Name)
}
