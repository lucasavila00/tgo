package compiler

import (
	"go/ast"
	"go/token"
)

func (e *loweringEmitter) switchStatement(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	node := operation.source.(*ast.SwitchStmt)
	target := output
	if operation.init != nil {
		wrapper := &ast.BlockStmt{}
		e.operations(operation.init, wrapper)
		node.Init = nil
		output.List = append(output.List, wrapper)
		target = wrapper
	}

	lowerCases := false
	for _, expression := range operation.expressions {
		if plannedExpressionHasWork(expression) {
			lowerCases = true
			break
		}
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
	target.List = append(target.List, node)
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
	output.List = append(output.List, node)
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
	for index, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		communication := operation.communications[index]
		if communication == nil || communication.channel == nil {
			continue
		}

		channel := e.emitTypedBind(
			communication.channelValue, communication.channel, output,
		)
		switch source := communication.source.(type) {
		case *ast.SendStmt:
			value := e.emitTypedBind(
				communication.sendValue, communication.value, output,
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
	output.List = append(output.List, node)
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
