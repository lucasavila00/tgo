package compiler

import (
	"go/ast"
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
				builtin.call.Fun = e.unit.generatedUniverse(builtin.name, builtin.position)
			}
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
				Lhs: []ast.Expr{e.valueName(operation.outputs[0], "operand")},
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

func (e *loweringEmitter) labeledStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.LabeledStmt)
	body := &ast.BlockStmt{}
	e.operations(operation.body, body)
	if len(body.List) == 0 {
		return
	}
	last := len(body.List) - 1
	switch body.List[last].(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
		*ast.TypeSwitchStmt, *ast.SelectStmt:
		body.List[last] = &ast.LabeledStmt{
			Label: node.Label, Colon: node.Colon, Stmt: body.List[last],
		}
		output.List = append(output.List, body.List...)
	default:
		node.Stmt = oneStatement(body.List)
		output.List = append(output.List, node)
	}
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
		assignment, ok := planned.source.(*ast.AssignStmt)
		if ok && len(planned.expressions) == 1 {
			assignment.Rhs = []ast.Expr{e.expression(planned.expressions[0], target)}
			node.Assign = assignment
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
	if operation.before != nil {
		e.operations(operation.before, output)
	}
	switch node := operation.source.(type) {
	case *ast.ReturnStmt:
		results := make([]ast.Expr, 0, len(operation.expressions))
		for _, expression := range operation.expressions {
			results = append(results, e.expressionResults(expression, output)...)
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
	output.List = append(output.List, &ast.AssignStmt{
		Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
	})
	return ast.NewIdent(name.Name)
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
			specification.Name = ast.NewIdent(e.fmtAlias)
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
