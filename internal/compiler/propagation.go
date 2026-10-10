package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

type propagationFunction struct {
	body       *ast.BlockStmt
	resultAST  []ast.Expr
	resultType *types.Tuple
	names      map[string]bool
}

type propagationLowerer struct {
	unit                *packageUnit
	source              *source
	function            propagationFunction
	fmtAlias            string
	gotos               map[string]bool
	names               map[string]bool
	inferredResultTypes map[types.Object]types.Type
	inferredResultNames map[string]types.Type
}

// lowerPropagations lowers postfix errors and comprehensions into direct control flow.
func (p *packageUnit) lowerPropagations() {
	for _, source := range p.Sources {
		functions := p.loweringFunctions(source)
		for _, function := range functions {
			lowerer := &propagationLowerer{
				unit: p, source: source, function: function, fmtAlias: "",
				gotos: functionGotoLabels(function.body), names: function.names,
				inferredResultTypes: make(map[types.Object]types.Type),
				inferredResultNames: make(map[string]types.Type),
			}
			function.body.List = lowerer.statements(function.body.List)
		}
		p.reportUnloweredExtensions(source)
	}
}

// functionGotoLabels finds source labels used by goto in one function.
// Nested functions have separate label scopes and separate lowerers.
func functionGotoLabels(body *ast.BlockStmt) map[string]bool {
	labels := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && function.Body != body {
			return false
		}
		branch, ok := node.(*ast.BranchStmt)
		if ok && branch.Tok == token.GOTO && branch.Label != nil {
			labels[branch.Label.Name] = true
		}
		return true
	})
	return labels
}

// loweringFunctions collects source functions but skips projection-only literals.
func (p *packageUnit) loweringFunctions(source *source) []propagationFunction {
	result := []propagationFunction(nil)
	ast.Inspect(source.File, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if _, _, found := comprehensionMarker(source, call); found {
				return false
			}
		}
		switch function := node.(type) {
		case *ast.FuncDecl:
			signature, _ := p.info.TypeOf(function.Name).(*types.Signature)
			result = appendLoweringFunction(result, function.Body, function.Type, signature)
		case *ast.FuncLit:
			signature, _ := p.info.TypeOf(function.Type).(*types.Signature)
			result = appendLoweringFunction(result, function.Body, function.Type, signature)
		}
		return true
	})
	return result
}

func appendLoweringFunction(
	functions []propagationFunction,
	body *ast.BlockStmt,
	typeNode *ast.FuncType,
	signature *types.Signature,
) []propagationFunction {
	if body == nil || signature == nil {
		return functions
	}
	return append(functions, propagationFunction{
		body:       body,
		resultAST:  flattenedResultTypes(typeNode.Results),
		resultType: signature.Results(),
		names:      functionNames(body, signature),
	})
}

func flattenedResultTypes(fields *ast.FieldList) []ast.Expr {
	if fields == nil {
		return nil
	}
	result := []ast.Expr(nil)
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, field.Type)
		}
	}
	return result
}

func (p *packageUnit) reportUnloweredExtensions(source *source) {
	ast.Inspect(source.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if metadata, found := propagationMarker(source, call); found {
			p.failAt(metadata.Bang, "error propagation needs a function body")
		}
		if metadata, _, found := comprehensionMarker(source, call); found {
			p.failAt(metadata.Position, "comprehension needs a function body")
		}
		return true
	})
	ast.Inspect(source.File, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if commas, found := source.FailureReturns[statement]; found {
			p.failAt(commas[0], "failure return needs a valid function signature")
		}
		return true
	})
}

func propagationMarker(
	source *source,
	expression ast.Expr,
) (propagationSource, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return propagationSource{}, false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return propagationSource{}, false
	}
	metadata, ok := source.Propagations[name.Name]
	return metadata, ok
}

func (l *propagationLowerer) statements(input []ast.Stmt) []ast.Stmt {
	result := make([]ast.Stmt, 0, len(input))
	for _, statement := range input {
		result = append(result, l.statement(statement)...)
	}
	return result
}

// scopedStatements restores inferred names when one lexical block ends.
func (l *propagationLowerer) scopedStatements(input []ast.Stmt) []ast.Stmt {
	outer := l.inferredResultNames
	l.inferredResultNames = cloneInferredResultNames(outer)
	result := l.statements(input)
	l.inferredResultNames = outer
	return result
}

func cloneInferredResultNames(input map[string]types.Type) map[string]types.Type {
	result := make(map[string]types.Type, len(input))
	for name, typ := range input {
		result[name] = typ
	}
	return result
}

//nolint:cyclop,gocognit // Each case preserves one Go statement evaluation rule.
func (l *propagationLowerer) statement(statement ast.Stmt) []ast.Stmt {
	switch node := statement.(type) {
	case *ast.BlockStmt:
		node.List = l.scopedStatements(node.List)
		return []ast.Stmt{node}
	case *ast.AssignStmt:
		l.rememberSimpleAssignmentTypes(node)
		return l.assignment(node)
	case *ast.IncDecStmt:
		return l.increment(node)
	case *ast.ExprStmt:
		return l.expressionStatement(node)
	case *ast.ReturnStmt:
		values, prefix := l.expressions(node.Results)
		node.Results = values
		if commas, ok := l.source.FailureReturns[node]; ok {
			delete(l.source.FailureReturns, node)
			return l.failureReturn(node, prefix, commas)
		}
		return append(prefix, node)
	case *ast.SendStmt:
		values, prefix := l.expressions([]ast.Expr{node.Chan, node.Value})
		node.Chan, node.Value = values[0], values[1]
		return append(prefix, node)
	case *ast.DeclStmt:
		return l.declaration(node)
	case *ast.IfStmt:
		outer := l.inferredResultNames
		l.inferredResultNames = cloneInferredResultNames(outer)
		scopedInitializer := node.Init != nil
		prefix := []ast.Stmt(nil)
		if l.statementHasLowering(node.Init) {
			prefix = l.simpleStatement(node.Init)
			node.Init = nil
		} else if assignment, ok := node.Init.(*ast.AssignStmt); ok {
			l.rememberSimpleAssignmentTypes(assignment)
		}
		node.Body.List = l.scopedStatements(node.Body.List)
		if node.Else != nil {
			rewritten := l.statement(node.Else)
			node.Else = oneStatement(rewritten)
		}
		condition, conditionPrefix := l.expression(node.Cond)
		node.Cond = condition
		if len(conditionPrefix) > 0 && node.Init != nil {
			prefix = append(prefix, l.simpleStatement(node.Init)...)
			node.Init = nil
		}
		prefix = append(prefix, conditionPrefix...)
		result := l.prefixedStatement(prefix, node, scopedInitializer)
		l.inferredResultNames = outer
		return result
	case *ast.RangeStmt:
		node.Body.List = l.scopedStatements(node.Body.List)
		value, prefix := l.expression(node.X)
		node.X = value
		return l.prefixedStatement(prefix, node, false)
	case *ast.SwitchStmt:
		outer := l.inferredResultNames
		l.inferredResultNames = cloneInferredResultNames(outer)
		scopedInitializer := node.Init != nil
		prefix := []ast.Stmt(nil)
		if l.statementHasLowering(node.Init) {
			prefix = l.simpleStatement(node.Init)
			node.Init = nil
		} else if assignment, ok := node.Init.(*ast.AssignStmt); ok {
			l.rememberSimpleAssignmentTypes(assignment)
		}
		l.caseBodies(node.Body)
		value, tagPrefix := l.optionalExpression(node.Tag)
		node.Tag = value
		if len(tagPrefix) > 0 && node.Init != nil {
			prefix = append(prefix, l.simpleStatement(node.Init)...)
			node.Init = nil
		}
		prefix = append(prefix, tagPrefix...)
		result := l.prefixedStatement(prefix, node, scopedInitializer)
		l.inferredResultNames = outer
		return result
	case *ast.TypeSwitchStmt:
		l.caseBodies(node.Body)
		l.missingStatementLowering(node.Init, "type switch initializer")
		l.missingStatementLowering(node.Assign, "type switch assignment")
		return []ast.Stmt{node}
	case *ast.ForStmt:
		node.Body.List = l.scopedStatements(node.Body.List)
		l.missingStatementLowering(node.Init, "for initializer")
		l.missingStatementLowering(node.Post, "for post statement")
		if l.hasLowering(node.Cond) {
			condition, prefix := l.expression(node.Cond)
			exit := &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: condition},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{
					Tok: token.BREAK,
				}}},
			}
			node.Cond = nil
			body := make([]ast.Stmt, 0, len(prefix)+1+len(node.Body.List))
			body = append(body, prefix...)
			body = append(body, exit)
			body = append(body, node.Body.List...)
			node.Body.List = body
		}
		return []ast.Stmt{node}
	case *ast.SelectStmt:
		for _, item := range node.Body.List {
			clause := item.(*ast.CommClause)
			l.missingStatementLowering(clause.Comm, "select communication")
			clause.Body = l.scopedStatements(clause.Body)
		}
		return []ast.Stmt{node}
	case *ast.GoStmt:
		if _, direct := propagationMarker(l.source, node.Call); direct {
			l.rejectExpression(node.Call, "go statement")
			return []ast.Stmt{node}
		}
		value, prefix := l.expression(node.Call)
		node.Call = value.(*ast.CallExpr)
		return append(prefix, node)
	case *ast.DeferStmt:
		if _, direct := propagationMarker(l.source, node.Call); direct {
			l.rejectExpression(node.Call, "defer statement")
			return []ast.Stmt{node}
		}
		value, prefix := l.expression(node.Call)
		node.Call = value.(*ast.CallExpr)
		return append(prefix, node)
	case *ast.LabeledStmt:
		return l.labeledStatement(node)
	default:
		return []ast.Stmt{statement}
	}
}

// labeledStatement keeps goto at the source label and moves loop or switch
// branches to a generated label when propagation adds a block.
func (l *propagationLowerer) labeledStatement(node *ast.LabeledStmt) []ast.Stmt {
	rewritten := oneStatement(l.statement(node.Stmt))
	block, ok := rewritten.(*ast.BlockStmt)
	if !ok || len(block.List) == 0 {
		node.Stmt = rewritten
		return []ast.Stmt{node}
	}

	last := len(block.List) - 1
	switch block.List[last].(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
		*ast.TypeSwitchStmt, *ast.SelectStmt:
	default:
		node.Stmt = rewritten
		return []ast.Stmt{node}
	}
	if !l.gotos[node.Label.Name] {
		block.List[last] = &ast.LabeledStmt{
			Label: node.Label,
			Colon: node.Colon,
			Stmt:  block.List[last],
		}
		return block.List
	}

	controlLabel := l.freshName("control").Name
	if !rewriteControlBranches(block.List[last], node.Label.Name, controlLabel) {
		node.Stmt = block
		return []ast.Stmt{node}
	}
	block.List[last] = &ast.LabeledStmt{
		Label: ast.NewIdent(controlLabel),
		Colon: node.Colon,
		Stmt:  block.List[last],
	}
	node.Stmt = block
	return []ast.Stmt{node}
}

// rewriteControlBranches preserves labeled break and continue statements.
// A function literal has its own label scope, so the walk does not enter it.
func rewriteControlBranches(node ast.Node, oldLabel string, newLabel string) bool {
	rewritten := false
	ast.Inspect(node, func(current ast.Node) bool {
		if _, ok := current.(*ast.FuncLit); ok {
			return false
		}
		branch, ok := current.(*ast.BranchStmt)
		if !ok || branch.Label == nil || branch.Label.Name != oldLabel {
			return true
		}
		if branch.Tok == token.BREAK || branch.Tok == token.CONTINUE {
			branch.Label = ast.NewIdent(newLabel)
			rewritten = true
		}
		return true
	})
	return rewritten
}

func (l *propagationLowerer) caseBodies(body *ast.BlockStmt) {
	for _, item := range body.List {
		clause := item.(*ast.CaseClause)
		for _, expression := range clause.List {
			l.missingExpressionLowering(expression, "switch case")
		}
		clause.Body = l.scopedStatements(clause.Body)
	}
}

func (l *propagationLowerer) simpleStatement(statement ast.Stmt) []ast.Stmt {
	if statement == nil {
		return nil
	}
	return l.statement(statement)
}

func (l *propagationLowerer) statementHasLowering(statement ast.Stmt) bool {
	if statement == nil {
		return false
	}
	found := false
	ast.Inspect(statement, func(node ast.Node) bool {
		if found {
			return false
		}
		if function, ok := node.(*ast.FuncLit); ok && function != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if ok {
			_, found = propagationMarker(l.source, call)
		}
		return !found
	})
	return found
}

func oneStatement(statements []ast.Stmt) ast.Stmt {
	if len(statements) == 1 {
		return statements[0]
	}
	return &ast.BlockStmt{List: statements}
}

// prefixedStatement adds a block only when source scope or goto rules need it.
func (l *propagationLowerer) prefixedStatement(
	prefix []ast.Stmt,
	statement ast.Stmt,
	scopedInitializer bool,
) []ast.Stmt {
	if len(prefix) == 0 {
		return []ast.Stmt{statement}
	}
	prefix = append(prefix, statement)
	if !scopedInitializer && len(l.gotos) == 0 {
		return prefix
	}
	return []ast.Stmt{&ast.BlockStmt{List: prefix}}
}

func (l *propagationLowerer) propagation(
	expression ast.Expr,
) ([]ast.Expr, []ast.Stmt) {
	metadata, ok := propagationMarker(l.source, expression)
	if !ok {
		return []ast.Expr{expression}, nil
	}
	marker := expression.(*ast.CallExpr)
	call, ok := unwrappedCompilerCall(marker.Args[0])
	if !ok {
		l.unit.failAt(metadata.Bang, "error propagation needs a call")
		return []ast.Expr{expression}, nil
	}
	signature := l.propagationCallSignature(call.Fun)
	ok = signature != nil
	if !ok || signature.Results().Len() == 0 ||
		!isPredeclaredError(signature.Results().At(signature.Results().Len()-1).Type()) {
		l.unit.failAt(metadata.Bang, "propagated call must end in the Go error type")
		return []ast.Expr{expression}, nil
	}
	if l.function.resultType.Len() == 0 ||
		!isPredeclaredError(l.function.resultType.At(l.function.resultType.Len()-1).Type()) {
		l.unit.failAt(metadata.Bang, "propagating function must end in the Go error type")
		return []ast.Expr{expression}, nil
	}
	loweredCall, prefix := l.expression(call)
	call = loweredCall.(*ast.CallExpr)
	values := make([]ast.Expr, 0, signature.Results().Len()-1)
	left := make([]ast.Expr, 0, signature.Results().Len())
	for range signature.Results().Len() - 1 {
		name := l.freshName("result")
		values = append(values, name)
		left = append(left, name)
	}
	errorName := l.freshName("err")
	left = append(left, errorName)
	assignment := &ast.AssignStmt{Lhs: left, Tok: token.DEFINE, Rhs: []ast.Expr{call}}
	prefix = append(prefix, assignment, l.errorBranch(metadata, errorName))
	return values, prefix
}

// propagationCallSignature gets a call type, including a receiver from an earlier propagation.
func (l *propagationLowerer) propagationCallSignature(
	function ast.Expr,
) *types.Signature {
	signature, _ := types.Unalias(l.unit.info.TypeOf(function)).(*types.Signature)
	if signature != nil {
		return signature
	}
	selector, ok := function.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	receiver := selector.X
	for {
		parentheses, wrapped := receiver.(*ast.ParenExpr)
		if !wrapped {
			break
		}
		receiver = parentheses.X
	}
	identifier, ok := receiver.(*ast.Ident)
	if !ok {
		return nil
	}
	receiverType := l.inferredResultTypes[l.unit.info.ObjectOf(identifier)]
	if receiverType == nil {
		receiverType = l.inferredResultNames[identifier.Name]
	}
	if receiverType == nil {
		return nil
	}
	object, _, _ := types.LookupFieldOrMethod(
		receiverType, true, l.unit.typed, selector.Sel.Name,
	)
	if object == nil {
		return nil
	}
	signature, _ = types.Unalias(object.Type()).(*types.Signature)
	return signature
}

func unwrappedCompilerCall(expression ast.Expr) (*ast.CallExpr, bool) {
	for {
		parentheses, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parentheses.X
	}
	call, ok := expression.(*ast.CallExpr)
	return call, ok
}

func isPredeclaredError(value types.Type) bool {
	object := types.Universe.Lookup("error")
	return object != nil && types.Identical(value, object.Type())
}

func (l *propagationLowerer) materialize(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil || !l.canMaterialize(expression) {
		return expression, nil
	}
	name := l.freshName("operand")
	statement := &ast.AssignStmt{
		Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
	}
	return name, []ast.Stmt{statement}
}

// materializeOrderedOperand saves computed values but leaves names and constants in place.
func (l *propagationLowerer) materializeOrderedOperand(
	expression ast.Expr,
) (ast.Expr, []ast.Stmt) {
	if isCheapAssignmentOperand(expression) {
		return expression, nil
	}
	return l.materialize(expression)
}

func (l *propagationLowerer) canMaterialize(expression ast.Expr) bool {
	value, ok := l.unit.info.Types[expression]
	if !ok || value.IsType() || value.Value != nil {
		return false
	}
	_, tuple := value.Type.(*types.Tuple)
	if tuple {
		return false
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		_, builtin := l.unit.info.Uses[identifier].(*types.Builtin)
		return !builtin && identifier.Name != "nil"
	}
	return true
}

func (l *propagationLowerer) rejectExpression(expression ast.Expr, context string) {
	l.diagnoseExpression(expression, context, false)
}

func (l *propagationLowerer) missingExpressionLowering(
	expression ast.Expr,
	context string,
) {
	l.diagnoseExpression(expression, context, true)
}

func (l *propagationLowerer) diagnoseExpression(
	expression ast.Expr,
	context string,
	missingLowering bool,
) {
	if expression == nil {
		return
	}
	ast.Inspect(expression, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if metadata, found := propagationMarker(l.source, call); found {
			message := "error propagation is not valid in %s"
			if missingLowering {
				message = "compiler does not yet lower error propagation in %s"
			}
			l.unit.failAt(metadata.Bang, message, context)
			return false
		}
		if metadata, _, found := comprehensionMarker(l.source, call); found {
			message := "comprehension is not valid in %s"
			if missingLowering {
				message = "compiler does not yet lower a comprehension in %s"
			}
			l.unit.failAt(metadata.Position, message, context)
			return false
		}
		return true
	})
}

func (l *propagationLowerer) missingStatementLowering(
	statement ast.Stmt,
	context string,
) {
	l.diagnoseStatement(statement, context, true)
}

func (l *propagationLowerer) diagnoseStatement(
	statement ast.Stmt,
	context string,
	missingLowering bool,
) {
	if statement == nil {
		return
	}
	ast.Inspect(statement, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if ok {
			l.diagnoseExpression(expression, context, missingLowering)
			return false
		}
		return true
	})
}
