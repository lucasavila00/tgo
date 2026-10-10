package compiler

import (
	"go/ast"
	"go/token"
	"strconv"
)

func (e *loweringEmitter) switchStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.SwitchStmt)
	lowerCases := plannedSwitchCasesHaveWork(operation)
	target := output
	if operation.init != nil && !lowerCases && !blockHasPlannedWork(operation.init) &&
		len(operation.init.operations) == 1 {
		node.Init = operation.init.operations[0].source
	} else if operation.init != nil {
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}
	if lowerCases && target == output {
		target = &ast.BlockStmt{}
		output.List = append(output.List, target)
	}

	if !lowerCases {
		e.emitDirectSwitch(operation, node, target)
		return
	}

	var tag *ast.Ident
	caseExpression := 0
	if node.Tag != nil {
		value := e.expression(operation.expressions[0], target)
		caseExpression++
		tag = e.freshName("tag")
		target.List = append(target.List, &ast.AssignStmt{
			Lhs: []ast.Expr{tag}, Tok: token.DEFINE, Rhs: []ast.Expr{value},
		})
	}

	selected := e.freshName("selected")
	target.List = append(target.List, &ast.AssignStmt{
		Lhs: []ast.Expr{selected}, Tok: token.DEFINE,
		Rhs: []ast.Expr{switchNegativeOne()},
	})
	for clauseIndex, item := range node.Body.List {
		clause := item.(*ast.CaseClause)
		for range clause.List {
			guarded := &ast.BlockStmt{}
			value := e.expression(operation.expressions[caseExpression], guarded)
			caseExpression++
			condition := value
			if tag != nil {
				condition = &ast.BinaryExpr{X: ast.NewIdent(tag.Name), Op: token.EQL, Y: value}
			}
			guarded.List = append(guarded.List, &ast.IfStmt{
				Cond: condition,
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent(selected.Name)}, Tok: token.ASSIGN,
					Rhs: []ast.Expr{switchIntegerLiteral(clauseIndex)},
				}}},
			})
			target.List = append(target.List, &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X: ast.NewIdent(selected.Name), Op: token.EQL, Y: switchNegativeOne(),
				},
				Body: guarded,
			})
		}
		if len(clause.List) != 0 {
			clause.List = []ast.Expr{switchIntegerLiteral(clauseIndex)}
		}
	}
	node.Tag = ast.NewIdent(selected.Name)
	e.emitSwitchBodies(operation, node)
	e.appendControl(target, operation.target, node)
}

func plannedSwitchCasesHaveWork(operation *plannedOperation) bool {
	for _, expression := range operation.expressions {
		if plannedExpressionHasWork(expression) {
			return true
		}
	}
	return false
}

func switchNegativeOne() ast.Expr {
	return &ast.UnaryExpr{Op: token.SUB, X: switchIntegerLiteral(1)}
}

func switchIntegerLiteral(value int) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)}
}

func (e *loweringEmitter) emitDirectSwitch(
	operation *plannedOperation,
	node *ast.SwitchStmt,
	output *ast.BlockStmt,
) {
	expression := 0
	if node.Tag != nil {
		node.Tag = e.expression(operation.expressions[0], output)
		expression++
	}
	for _, item := range node.Body.List {
		clause := item.(*ast.CaseClause)
		for index := range clause.List {
			clause.List[index] = e.expression(operation.expressions[expression], output)
			expression++
		}
	}
	e.emitSwitchBodies(operation, node)
	e.appendControl(output, operation.target, node)
}

func (e *loweringEmitter) emitSwitchBodies(
	operation *plannedOperation,
	node *ast.SwitchStmt,
) {
	for index, item := range node.Body.List {
		clause := item.(*ast.CaseClause)
		body := &ast.BlockStmt{}
		e.operations(operation.cases[index], body)
		clause.Body = body.List
	}
}

func (e *loweringEmitter) selectStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.SelectStmt)
	target := &ast.BlockStmt{}
	output.List = append(output.List, target)
	for index, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		communication := operation.communications[index]
		if communication == nil || communication.channel == nil {
			continue
		}

		channel := e.emitTypedBind(
			communication.channelValue, communication.channel, target,
		)
		switch source := communication.source.(type) {
		case *ast.SendStmt:
			value := e.emitTypedBind(
				communication.sendValue, communication.value, target,
			)
			clause.Comm = &ast.SendStmt{Chan: channel, Arrow: source.Arrow, Value: value}
		case *ast.ExprStmt:
			receive := selectReceiveExpression(source.X)
			clause.Comm = &ast.ExprStmt{X: &ast.UnaryExpr{
				OpPos: receive.OpPos, Op: token.ARROW, X: channel,
			}}
		case *ast.AssignStmt:
			receive := selectReceiveExpression(source.Rhs[0])
			left := make([]ast.Expr, 0, len(communication.receiveValues))
			for _, value := range communication.receiveValues {
				left = append(left, e.valueName(value.id, "received"))
			}
			clause.Comm = &ast.AssignStmt{
				Lhs: left, TokPos: source.TokPos, Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.UnaryExpr{
					OpPos: receive.OpPos, Op: token.ARROW, X: channel,
				}},
			}
		}
	}

	for index, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		body := &ast.BlockStmt{}
		e.emitReceiveStore(operation.communications[index], body)
		e.operations(operation.cases[index], body)
		clause.Body = body.List
	}
	e.appendControl(target, operation.target, node)
}

func selectReceiveExpression(expression ast.Expr) *ast.UnaryExpr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression.(*ast.UnaryExpr)
		}
		expression = parenthesized.X
	}
}
