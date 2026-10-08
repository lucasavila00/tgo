package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

type propagationFunction struct {
	body       *ast.BlockStmt
	resultAST  []ast.Expr
	resultType *types.Tuple
}

type propagationLowerer struct {
	unit     *packageUnit
	source   *source
	function propagationFunction
	fmtAlias string
	gotos    map[string]bool
}

// lowerPropagations turns each postfix marker into a direct Go error branch.
func (p *packageUnit) lowerPropagations() {
	for _, source := range p.Sources {
		functions := p.propagationFunctions(source.File)
		for _, function := range functions {
			lowerer := &propagationLowerer{
				unit: p, source: source, function: function, fmtAlias: "",
				gotos: functionGotoLabels(function.body),
			}
			function.body.List = lowerer.statements(function.body.List)
		}
		p.reportUnloweredPropagations(source)
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

// propagationFunctions collects named and literal functions before the AST changes.
func (p *packageUnit) propagationFunctions(file *ast.File) []propagationFunction {
	result := []propagationFunction(nil)
	ast.Inspect(file, func(node ast.Node) bool {
		switch function := node.(type) {
		case *ast.FuncDecl:
			signature, _ := p.info.TypeOf(function.Name).(*types.Signature)
			result = appendPropagationFunction(result, function.Body, function.Type, signature)
		case *ast.FuncLit:
			signature, _ := p.info.TypeOf(function.Type).(*types.Signature)
			result = appendPropagationFunction(result, function.Body, function.Type, signature)
		}
		return true
	})
	return result
}

func appendPropagationFunction(
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

func (p *packageUnit) reportUnloweredPropagations(source *source) {
	ast.Inspect(source.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if metadata, found := propagationMarker(source, call); found {
			p.failAt(metadata.Bang, "error propagation needs a function body")
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

//nolint:cyclop,gocognit // Each case preserves one Go statement evaluation rule.
func (l *propagationLowerer) statement(statement ast.Stmt) []ast.Stmt {
	switch node := statement.(type) {
	case *ast.BlockStmt:
		node.List = l.statements(node.List)
		return []ast.Stmt{node}
	case *ast.AssignStmt:
		return l.assignment(node)
	case *ast.IncDecStmt:
		return l.increment(node)
	case *ast.ExprStmt:
		return l.expressionStatement(node)
	case *ast.ReturnStmt:
		values, prefix := l.expressions(node.Results)
		node.Results = values
		return append(prefix, node)
	case *ast.SendStmt:
		values, prefix := l.expressions([]ast.Expr{node.Chan, node.Value})
		node.Chan, node.Value = values[0], values[1]
		return append(prefix, node)
	case *ast.DeclStmt:
		return l.declaration(node)
	case *ast.IfStmt:
		node.Body.List = l.statements(node.Body.List)
		if node.Else != nil {
			rewritten := l.statement(node.Else)
			node.Else = oneStatement(rewritten)
		}
		condition, prefix := l.expression(node.Cond)
		node.Cond = condition
		if len(prefix) > 0 || l.statementHasPropagation(node.Init) {
			prefix = append(l.simpleStatement(node.Init), prefix...)
			node.Init = nil
		}
		return wrapStatement(prefix, node)
	case *ast.RangeStmt:
		node.Body.List = l.statements(node.Body.List)
		value, prefix := l.expression(node.X)
		node.X = value
		return wrapStatement(prefix, node)
	case *ast.SwitchStmt:
		l.caseBodies(node.Body)
		value, prefix := l.optionalExpression(node.Tag)
		node.Tag = value
		if len(prefix) > 0 || l.statementHasPropagation(node.Init) {
			prefix = append(l.simpleStatement(node.Init), prefix...)
			node.Init = nil
		}
		return wrapStatement(prefix, node)
	case *ast.TypeSwitchStmt:
		l.caseBodies(node.Body)
		l.rejectStatement(node.Init, "type switch initializer")
		l.rejectStatement(node.Assign, "type switch assignment")
		return []ast.Stmt{node}
	case *ast.ForStmt:
		node.Body.List = l.statements(node.Body.List)
		l.rejectStatement(node.Init, "for initializer")
		l.rejectStatement(node.Post, "for post statement")
		if l.hasPropagation(node.Cond) {
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
			l.rejectStatement(clause.Comm, "select communication")
			clause.Body = l.statements(clause.Body)
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
		return []ast.Stmt{block}
	}

	controlLabel := l.unit.freshIdentifier("__tgo_control")
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

func (l *propagationLowerer) assignment(node *ast.AssignStmt) []ast.Stmt {
	if !l.assignmentHasPropagation(node) {
		return []ast.Stmt{node}
	}
	left, prefix := l.assignmentTargets(node.Lhs)
	node.Lhs = left
	if len(node.Rhs) == 1 {
		if metadata, ok := propagationMarker(l.source, node.Rhs[0]); ok {
			values, before := l.propagation(node.Rhs[0])
			prefix = append(prefix, before...)
			if len(values) != len(node.Lhs) {
				l.reportResultCount(metadata, len(values), len(node.Lhs), "assignment")
			}
			node.Rhs = values
			return append(prefix, node)
		}
	}
	values, before := l.expressions(node.Rhs)
	prefix = append(prefix, before...)
	node.Rhs = values
	return append(prefix, node)
}

func (l *propagationLowerer) assignmentHasPropagation(node *ast.AssignStmt) bool {
	for _, expression := range append(append([]ast.Expr(nil), node.Lhs...), node.Rhs...) {
		if l.hasPropagation(expression) {
			return true
		}
	}
	return false
}

// assignmentTargets evaluates target operands before a moved right side.
// It follows Go assignment order without copying an array target.
func (l *propagationLowerer) assignmentTargets(
	targets []ast.Expr,
) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), targets...)
	prefix := []ast.Stmt(nil)
	for index, sourceTarget := range result {
		target, targetPrefix := l.assignmentTarget(sourceTarget)
		result[index] = target
		prefix = append(prefix, targetPrefix...)
	}
	return result, prefix
}

func (l *propagationLowerer) assignmentTarget(
	target ast.Expr,
) (ast.Expr, []ast.Stmt) {
	switch node := target.(type) {
	case *ast.ParenExpr:
		value, prefix := l.assignmentTarget(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.cheapAssignmentOperand(node.X)
		node.X = value
		return node, prefix
	case *ast.SelectorExpr:
		selection := l.unit.info.Selections[node]
		var value ast.Expr
		var prefix []ast.Stmt
		if selection != nil && selection.Indirect() {
			value, prefix = l.cheapAssignmentOperand(node.X)
		} else {
			value, prefix = l.assignmentTarget(node.X)
		}
		node.X = value
		return node, prefix
	case *ast.IndexExpr:
		var value ast.Expr
		var prefix []ast.Stmt
		if admitsArray(l.unit.info.TypeOf(node.X)) {
			value, prefix = l.assignmentTarget(node.X)
		} else {
			value, prefix = l.cheapAssignmentOperand(node.X)
		}
		index, indexPrefix := l.cheapAssignmentOperand(node.Index)
		node.X, node.Index = value, index
		resultPrefix := make([]ast.Stmt, 0, len(prefix)+len(indexPrefix))
		resultPrefix = append(resultPrefix, prefix...)
		resultPrefix = append(resultPrefix, indexPrefix...)
		return node, resultPrefix
	default:
		return target, nil
	}
}

// admitsArray reports whether a type or type set has an array term.
func admitsArray(value types.Type) bool {
	if value == nil {
		return false
	}
	value = types.Unalias(value)
	switch node := value.(type) {
	case *types.Array:
		return true
	case *types.Named:
		return admitsArray(node.Underlying())
	case *types.TypeParam:
		return admitsArray(node.Constraint())
	case *types.Interface:
		node.Complete()
		for index := range node.NumEmbeddeds() {
			if admitsArray(node.EmbeddedType(index)) {
				return true
			}
		}
	case *types.Union:
		for index := range node.Len() {
			if admitsArray(node.Term(index).Type()) {
				return true
			}
		}
	}
	return false
}

func (l *propagationLowerer) cheapAssignmentOperand(
	expression ast.Expr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.expression(expression)
	if isCheapAssignmentOperand(value) {
		return value, prefix
	}
	value, before := l.materialize(value)
	return value, append(prefix, before...)
}

func isCheapAssignmentOperand(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	default:
		return false
	}
}

func (l *propagationLowerer) increment(node *ast.IncDecStmt) []ast.Stmt {
	if !l.hasPropagation(node.X) {
		return []ast.Stmt{node}
	}
	value, prefix := l.assignmentTarget(node.X)
	node.X = value
	return append(prefix, node)
}

func (l *propagationLowerer) expressionStatement(node *ast.ExprStmt) []ast.Stmt {
	if metadata, ok := propagationMarker(l.source, node.X); ok {
		values, prefix := l.propagation(node.X)
		if len(values) != 0 {
			l.unit.failAt(metadata.Bang, "discarded propagated call returns values")
		}
		return prefix
	}
	value, prefix := l.expression(node.X)
	node.X = value
	return append(prefix, node)
}

func (l *propagationLowerer) declaration(node *ast.DeclStmt) []ast.Stmt {
	general, ok := node.Decl.(*ast.GenDecl)
	if !ok {
		return []ast.Stmt{node}
	}
	prefix := []ast.Stmt(nil)
	for _, item := range general.Specs {
		value, ok := item.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if len(value.Values) == 1 {
			if metadata, direct := propagationMarker(l.source, value.Values[0]); direct {
				values, before := l.propagation(value.Values[0])
				if len(values) != len(value.Names) {
					l.reportResultCount(metadata, len(values), len(value.Names), "declaration")
				}
				value.Values = values
				prefix = append(prefix, before...)
				continue
			}
		}
		values, before := l.expressions(value.Values)
		value.Values = values
		prefix = append(prefix, before...)
	}
	return append(prefix, node)
}

func (l *propagationLowerer) reportResultCount(
	metadata propagationSource,
	results int,
	targets int,
	context string,
) {
	l.unit.failAt(
		metadata.Bang,
		"propagated call returns %d values; %s needs %d",
		results,
		context,
		targets,
	)
}

func (l *propagationLowerer) caseBodies(body *ast.BlockStmt) {
	for _, item := range body.List {
		clause := item.(*ast.CaseClause)
		for _, expression := range clause.List {
			l.rejectExpression(expression, "switch case")
		}
		clause.Body = l.statements(clause.Body)
	}
}

func (l *propagationLowerer) simpleStatement(statement ast.Stmt) []ast.Stmt {
	if statement == nil {
		return nil
	}
	return l.statement(statement)
}

func (l *propagationLowerer) statementHasPropagation(statement ast.Stmt) bool {
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

func wrapStatement(prefix []ast.Stmt, statement ast.Stmt) []ast.Stmt {
	if len(prefix) == 0 {
		return []ast.Stmt{statement}
	}
	prefix = append(prefix, statement)
	return []ast.Stmt{&ast.BlockStmt{List: prefix}}
}

func (l *propagationLowerer) expressions(input []ast.Expr) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), input...)
	prefix := []ast.Stmt(nil)
	for index, expression := range result {
		lowered, before := l.expression(expression)
		prefix = append(prefix, before...)
		if l.laterPropagation(result[index+1:]) {
			lowered, before = l.materializeOrderedOperand(lowered)
			prefix = append(prefix, before...)
		}
		result[index] = lowered
	}
	return result, prefix
}

func (l *propagationLowerer) laterPropagation(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		if l.hasPropagation(expression) {
			return true
		}
	}
	return false
}

//nolint:cyclop,gocognit // Each expression case keeps Go operand order.
func (l *propagationLowerer) expression(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil || !l.hasPropagation(expression) {
		return expression, nil
	}
	if metadata, ok := propagationMarker(l.source, expression); ok {
		values, prefix := l.propagation(expression)
		if len(values) != 1 {
			l.unit.failAt(metadata.Bang, "nested propagated call needs one value")
			return expression, prefix
		}
		return values[0], prefix
	}
	switch node := expression.(type) {
	case *ast.CallExpr:
		ordered := append([]ast.Expr{node.Fun}, node.Args...)
		ordered, prefix := l.expressions(ordered)
		node.Fun, node.Args = ordered[0], ordered[1:]
		return node, prefix
	case *ast.BinaryExpr:
		if node.Op == token.LAND || node.Op == token.LOR {
			return l.shortCircuit(node)
		}
		values, prefix := l.expressions([]ast.Expr{node.X, node.Y})
		node.X, node.Y = values[0], values[1]
		return node, prefix
	case *ast.IndexExpr:
		return l.indexExpression(node)
	case *ast.IndexListExpr:
		values := append([]ast.Expr{node.X}, node.Indices...)
		values, prefix := l.expressions(values)
		node.X, node.Indices = values[0], values[1:]
		return node, prefix
	case *ast.SelectorExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.SliceExpr:
		return l.sliceExpression(node)
	case *ast.ParenExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.UnaryExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.TypeAssertExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.CompositeLit:
		values, prefix := l.expressions(node.Elts)
		node.Elts = values
		return node, prefix
	case *ast.KeyValueExpr:
		values, prefix := l.expressions([]ast.Expr{node.Key, node.Value})
		node.Key, node.Value = values[0], values[1]
		return node, prefix
	case *ast.FuncLit:
		return node, nil
	default:
		l.rejectExpression(expression, "this expression")
		return expression, nil
	}
}

// indexExpression keeps an array path addressable while it lowers the index.
func (l *propagationLowerer) indexExpression(
	node *ast.IndexExpr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.indexedOperand(node.X)
	index, indexPrefix := l.expression(node.Index)
	node.X, node.Index = value, index
	return node, append(prefix, indexPrefix...)
}

// sliceExpression keeps an array path addressable while it lowers each bound.
func (l *propagationLowerer) sliceExpression(
	node *ast.SliceExpr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.indexedOperand(node.X)
	bounds, boundsPrefix := l.expressions([]ast.Expr{node.Low, node.High, node.Max})
	node.X, node.Low, node.High, node.Max = value, bounds[0], bounds[1], bounds[2]
	return node, append(prefix, boundsPrefix...)
}

// indexedOperand avoids an array copy but saves other computed containers.
func (l *propagationLowerer) indexedOperand(
	expression ast.Expr,
) (ast.Expr, []ast.Stmt) {
	if !admitsArray(l.unit.info.TypeOf(expression)) {
		return l.cheapAssignmentOperand(expression)
	}
	switch expression.(type) {
	case *ast.Ident, *ast.ParenExpr, *ast.StarExpr,
		*ast.SelectorExpr, *ast.IndexExpr:
		return l.assignmentTarget(expression)
	default:
		return l.cheapAssignmentOperand(expression)
	}
}

// shortCircuit keeps conditional right-side evaluation around propagation branches.
func (l *propagationLowerer) shortCircuit(node *ast.BinaryExpr) (ast.Expr, []ast.Stmt) {
	left, prefix := l.expression(node.X)
	leftName := ast.NewIdent(l.unit.freshIdentifier("__tgo_left"))
	prefix = append(prefix, &ast.AssignStmt{
		Lhs: []ast.Expr{leftName}, Tok: token.DEFINE, Rhs: []ast.Expr{left},
	})
	defaultValue := "false"
	condition := ast.Expr(leftName)
	if node.Op == token.LOR {
		defaultValue = "true"
		condition = &ast.UnaryExpr{Op: token.NOT, X: leftName}
	}
	resultName := ast.NewIdent(l.unit.freshIdentifier("__tgo_condition"))
	prefix = append(prefix, &ast.AssignStmt{
		Lhs: []ast.Expr{resultName},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{l.unit.generatedUniverse(defaultValue, node.OpPos)},
	})
	right, rightPrefix := l.expression(node.Y)
	rightPrefix = append(rightPrefix, &ast.AssignStmt{
		Lhs: []ast.Expr{resultName}, Tok: token.ASSIGN, Rhs: []ast.Expr{right},
	})
	prefix = append(prefix, &ast.IfStmt{
		Cond: condition,
		Body: &ast.BlockStmt{List: rightPrefix},
	})
	return resultName, prefix
}

func (l *propagationLowerer) optionalExpression(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil {
		return nil, nil
	}
	return l.expression(expression)
}

func (l *propagationLowerer) hasPropagation(expression ast.Expr) bool {
	if expression == nil {
		return false
	}
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested && node != expression {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if _, ok := propagationMarker(l.source, call); ok {
			found = true
			return false
		}
		return true
	})
	return found
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
	signature, ok := types.Unalias(l.unit.info.TypeOf(call.Fun)).(*types.Signature)
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
		name := ast.NewIdent(l.unit.freshIdentifier("__tgo_value"))
		values = append(values, name)
		left = append(left, name)
	}
	errorName := ast.NewIdent(l.unit.freshIdentifier("__tgo_error"))
	left = append(left, errorName)
	assignment := &ast.AssignStmt{Lhs: left, Tok: token.DEFINE, Rhs: []ast.Expr{call}}
	prefix = append(prefix, assignment, l.errorBranch(metadata, errorName))
	return values, prefix
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

func (l *propagationLowerer) errorBranch(
	metadata propagationSource,
	errorName *ast.Ident,
) ast.Stmt {
	zeroNames := make([]ast.Expr, 0, len(l.function.resultAST)-1)
	body := make([]ast.Stmt, 0, len(l.function.resultAST))
	for _, resultType := range l.function.resultAST[:len(l.function.resultAST)-1] {
		name := ast.NewIdent(l.unit.freshIdentifier("__tgo_zero"))
		specification := &ast.ValueSpec{Names: []*ast.Ident{name}, Type: resultType}
		l.unit.generatedValues[specification] = true
		declaration := &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok:   token.VAR,
			Specs: []ast.Spec{specification},
		}}
		body = append(body, declaration)
		zeroNames = append(zeroNames, name)
	}
	formatError := l.unit.generatedObject(
		l.formatQualifier(), "fmt", "Errorf", metadata.Bang,
	)
	wrapped := call(formatError,
		&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(metadata.Name + ": %w")},
		errorName,
	)
	results := make([]ast.Expr, 0, len(zeroNames)+1)
	results = append(results, zeroNames...)
	results = append(results, wrapped)
	body = append(body, &ast.ReturnStmt{Results: results})
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: errorName, Op: token.NEQ,
			Y: l.unit.generatedUniverse("nil", metadata.Bang),
		},
		Body: &ast.BlockStmt{List: body},
	}
}

func (l *propagationLowerer) formatQualifier() string {
	if l.fmtAlias != "" {
		return l.fmtAlias
	}
	for _, specification := range l.source.File.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "fmt" {
			continue
		}
		if specification.Name == nil {
			l.fmtAlias = "fmt"
			return l.fmtAlias
		}
		if specification.Name.Name == "_" {
			l.fmtAlias = l.unit.freshIdentifier("__tgo_fmt")
			specification.Name = ast.NewIdent(l.fmtAlias)
			return l.fmtAlias
		}
		if specification.Name.Name == "." {
			return ""
		}
		l.fmtAlias = specification.Name.Name
		return l.fmtAlias
	}
	l.fmtAlias = l.unit.freshIdentifier("__tgo_fmt")
	astutil.AddNamedImport(l.unit.fs, l.source.File, l.fmtAlias, "fmt")
	return l.fmtAlias
}

func (l *propagationLowerer) materialize(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil || !l.canMaterialize(expression) {
		return expression, nil
	}
	name := ast.NewIdent(l.unit.freshIdentifier("__tgo_operand"))
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
	if expression == nil {
		return
	}
	ast.Inspect(expression, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if metadata, found := propagationMarker(l.source, call); found {
			l.unit.failAt(metadata.Bang, "error propagation is not valid in %s", context)
			return false
		}
		return true
	})
}

func (l *propagationLowerer) rejectStatement(statement ast.Stmt, context string) {
	if statement == nil {
		return
	}
	ast.Inspect(statement, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if ok {
			l.rejectExpression(expression, context)
			return false
		}
		return true
	})
}
