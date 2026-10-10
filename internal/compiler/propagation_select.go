package compiler

import (
	"go/ast"
	"go/token"
)

func (l *propagationLowerer) selectStatement(node *ast.SelectStmt) []ast.Stmt {
	lowerOperands := l.selectOperandsHavePropagation(node.Body)
	prefix := []ast.Stmt(nil)
	for _, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		bodyPrefix := []ast.Stmt(nil)
		if lowerOperands {
			prefix = append(prefix, l.lowerSelectOperands(clause.Comm)...)
		}
		if assignment, ok := clause.Comm.(*ast.AssignStmt); ok &&
			l.expressionsHaveLowering(assignment.Lhs) {
			bodyPrefix = l.lowerSelectReceiveTargets(assignment)
		}
		l.missingStatementLowering(clause.Comm, "select communication")
		bodyPrefix = append(bodyPrefix, clause.Body...)
		clause.Body = l.scopedStatements(bodyPrefix)
	}
	return l.prefixedStatement(prefix, node, false)
}

func (l *propagationLowerer) selectOperandsHavePropagation(body *ast.BlockStmt) bool {
	for _, item := range body.List {
		clause := item.(*ast.CommClause)
		switch communication := clause.Comm.(type) {
		case *ast.SendStmt:
			if l.hasLowering(communication.Chan) || l.hasLowering(communication.Value) {
				return true
			}
		case *ast.ExprStmt:
			if receive, ok := communication.X.(*ast.UnaryExpr); ok &&
				receive.Op == token.ARROW && l.hasLowering(receive.X) {
				return true
			}
		case *ast.AssignStmt:
			if receive := selectReceive(communication); receive != nil &&
				l.hasLowering(receive.X) {
				return true
			}
		}
	}
	return false
}

func (l *propagationLowerer) lowerSelectOperands(communication ast.Stmt) []ast.Stmt {
	switch node := communication.(type) {
	case *ast.SendStmt:
		channel, channelPrefix := l.selectOperand(node.Chan)
		value, valuePrefix := l.selectOperand(node.Value)
		node.Chan, node.Value = channel, value
		return append(channelPrefix, valuePrefix...)
	case *ast.ExprStmt:
		receive, ok := node.X.(*ast.UnaryExpr)
		if !ok || receive.Op != token.ARROW {
			return nil
		}
		channel, prefix := l.selectOperand(receive.X)
		receive.X = channel
		return prefix
	case *ast.AssignStmt:
		receive := selectReceive(node)
		if receive == nil {
			return nil
		}
		channel, prefix := l.selectOperand(receive.X)
		receive.X = channel
		return prefix
	default:
		return nil
	}
}

func (l *propagationLowerer) selectOperand(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	value, prefix := l.expression(expression)
	value, stored := l.materialize(value)
	return value, append(prefix, stored...)
}

func selectReceive(assignment *ast.AssignStmt) *ast.UnaryExpr {
	if len(assignment.Rhs) != 1 {
		return nil
	}
	receive, _ := assignment.Rhs[0].(*ast.UnaryExpr)
	if receive == nil || receive.Op != token.ARROW {
		return nil
	}
	return receive
}

func (l *propagationLowerer) expressionsHaveLowering(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		if l.hasLowering(expression) {
			return true
		}
	}
	return false
}

func (l *propagationLowerer) lowerSelectReceiveTargets(
	assignment *ast.AssignStmt,
) []ast.Stmt {
	receive := selectReceive(assignment)
	values := make([]ast.Expr, len(assignment.Lhs))
	targets := make([]ast.Expr, len(assignment.Lhs))
	for index := range values {
		name := l.freshName("received")
		values[index] = ast.NewIdent(name.Name)
		targets[index] = name
	}
	moved := &ast.AssignStmt{
		Lhs: assignment.Lhs, Tok: assignment.Tok, Rhs: values,
	}
	assignment.Lhs = targets
	assignment.Tok = token.DEFINE
	assignment.Rhs = []ast.Expr{receive}
	return l.assignment(moved)
}
