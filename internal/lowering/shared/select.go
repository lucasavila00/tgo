package shared

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

// selectStatement lowers select operands in source order before selection and retains the chosen case body.
func (r *Context) selectStatement(s *ast.SelectStmt) []ast.Stmt {
	var roots []ast.Node
	for _, statement := range s.Body.List {
		roots = append(roots, statement.(*ast.CommClause).Comm)
	}
	exit := r.prepareBreakExit(s, roots...)

	communicationChanged := false
	for _, statement := range s.Body.List {
		if r.Contains(statement.(*ast.CommClause).Comm) {
			communicationChanged = true
		}
	}

	output := *s
	output.Body = &ast.BlockStmt{}
	var statements []ast.Stmt
	for _, statement := range s.Body.List {
		original := statement.(*ast.CommClause)
		clauseStatements, clause := r.selectClause(original, communicationChanged)
		statements = append(statements, clauseStatements...)
		output.Body.List = append(output.Body.List, clause)
	}
	return r.withBreakExit(s, &output, append(statements, &output), exit)
}

// selectClause returns operand setup, copied communication, and the body that runs after this case is selected.
func (r *Context) selectClause(original *ast.CommClause, communicationChanged bool) ([]ast.Stmt,
	*ast.CommClause) {
	clause := *original
	var statements []ast.Stmt
	var body []ast.Stmt
	if communicationChanged && original.Comm != nil {
		switch communication := original.Comm.(type) {
		case *ast.SendStmt:
			statements, clause.Comm = r.selectSend(communication)
		case *ast.ExprStmt:
			statements, clause.Comm, body = r.selectReceiveExpression(communication)
		case *ast.AssignStmt:
			statements, clause.Comm, body = r.selectReceiveAssignment(communication)
		}
	}
	for _, statement := range original.Body {
		body = append(body, r.Statement(statement)...)
	}
	clause.Body = body
	return statements, &clause
}

// selectSend captures the channel and sent value before selection and builds a send using those saved operands.
func (r *Context) selectSend(original *ast.SendStmt) ([]ast.Stmt, ast.Stmt) {
	channelStatements, channel := r.Capture(original.Chan)
	valueStatements, value := r.Capture(original.Value)
	statements := append(channelStatements, valueStatements...)
	send := *original
	send.Chan = channel
	send.Value = value
	return statements, &send
}

// selectReceiveExpression captures the channel before selection and places an inserted action after the receive.
func (r *Context) selectReceiveExpression(original *ast.ExprStmt) ([]ast.Stmt, ast.Stmt, []ast.Stmt) {
	receive := ast.Unparen(original.X).(*ast.UnaryExpr)
	channelStatements, channel := r.Capture(receive.X)
	receiveExpression := *receive
	receiveExpression.X = channel
	if receive != ast.Unparen(r.input.Target) {
		return channelStatements, &ast.ExprStmt{X: &receiveExpression}, nil
	}

	r.request.inserted = true
	value := r.Name()
	statements := append(channelStatements, Variable(value, r.TemporaryType(r.TypeOf(receive), receive), nil))
	communication := &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN,
		Rhs: []ast.Expr{&receiveExpression}}
	body := r.input.After([]ast.Expr{value})
	body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN,
		Rhs: []ast.Expr{value}})
	return statements, communication, body
}

// selectReceiveAssignment receives into fresh locals and evaluates destinations only in the selected case.
func (r *Context) selectReceiveAssignment(original *ast.AssignStmt) ([]ast.Stmt, ast.Stmt, []ast.Stmt) {
	receive := ast.Unparen(original.Rhs[0]).(*ast.UnaryExpr)
	statements, channel := r.Capture(receive.X)
	receiveExpression := *receive
	receiveExpression.X = channel

	values := make([]ast.Expr, len(original.Lhs))
	for i := range values {
		value := r.Name()
		values[i] = value
		valueType := r.selectReceiveType(receive, i)
		statements = append(statements, Variable(value, r.TemporaryType(valueType, receive), nil))
	}
	communication := &ast.AssignStmt{Lhs: values, Tok: token.ASSIGN, Rhs: []ast.Expr{&receiveExpression}}

	var body []ast.Stmt
	if receive == ast.Unparen(r.input.Target) {
		r.request.inserted = true
		body = append(body, r.input.After(values)...)
	}
	store := *original
	store.Rhs = values
	if original.Tok == token.ASSIGN {
		store.Lhs = make([]ast.Expr, len(original.Lhs))
		for i, left := range original.Lhs {
			placeStatements, destination := r.place(left)
			body = append(body, placeStatements...)
			store.Lhs[i] = destination
		}
	}
	for _, destination := range store.Lhs {
		if r.deferredPlace(destination) {
			body = append(body, r.orderedStores(store.Lhs, store.Rhs, store.Tok)...)
			return statements, communication, body
		}
	}
	body = append(body, &store)
	return statements, communication, body
}

// selectReceiveType reads a receive result's checked type, including the Boolean comma-ok result.
func (r *Context) selectReceiveType(receive *ast.UnaryExpr, index int) types.Type {
	typ := r.TypeOf(receive)
	if tuple, ok := typ.(*types.Tuple); ok {
		return tuple.At(index).Type()
	}
	if index == 1 {
		return types.Typ[types.Bool]
	}
	return typ
}
