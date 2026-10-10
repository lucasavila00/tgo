package compiler

import (
	"go/ast"
	"go/token"
)

// directShortAssignment keeps source names in the propagated Go call.
func (l *propagationLowerer) directShortAssignment(
	node *ast.AssignStmt,
) ([]ast.Stmt, bool) {
	if node.Tok != token.DEFINE || len(node.Rhs) != 1 || !identifierTargets(node.Lhs) {
		return nil, false
	}
	call, statements, callIndex, direct := l.directCall(
		node.Rhs[0], len(node.Lhs), "assignment",
	)
	if !direct {
		return nil, false
	}
	if call == nil {
		return append(statements, node), true
	}
	node.Lhs = append(node.Lhs, call.Lhs[len(call.Lhs)-1])
	node.Rhs = call.Rhs
	statements[callIndex] = node
	return statements, true
}

// directVariableDeclaration keeps an inferred var declaration as one Go call.
func (l *propagationLowerer) directVariableDeclaration(
	node *ast.DeclStmt,
	general *ast.GenDecl,
) ([]ast.Stmt, bool) {
	if general.Tok != token.VAR || len(general.Specs) != 1 {
		return nil, false
	}
	value, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || value.Type != nil || len(value.Values) != 1 {
		return nil, false
	}
	call, statements, callIndex, direct := l.directCall(
		value.Values[0], len(value.Names), "declaration",
	)
	if !direct {
		return nil, false
	}
	if call == nil {
		return append(statements, node), true
	}
	errorName, ok := call.Lhs[len(call.Lhs)-1].(*ast.Ident)
	if !ok {
		return append(statements, node), true
	}
	errorName.NamePos = value.Names[len(value.Names)-1].End()
	value.Names = append(value.Names, errorName)
	value.Values = call.Rhs
	statements[callIndex] = node
	if value.Comment != nil {
		for _, statement := range statements[callIndex+1:] {
			positionGeneratedStatement(statement, value.Comment.End()+1)
		}
	}
	return statements, true
}

func positionGeneratedStatement(statement ast.Stmt, position token.Pos) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		positionGeneratedAssignment(node, position)
	case *ast.DeclStmt:
		if declaration, ok := node.Decl.(*ast.GenDecl); ok {
			positionGeneratedDeclaration(declaration, position)
		}
	case *ast.IfStmt:
		positionGeneratedIf(node, position)
	}
}

func positionGeneratedAssignment(statement *ast.AssignStmt, position token.Pos) {
	if statement.TokPos == token.NoPos {
		statement.TokPos = position
	}
	for _, expression := range statement.Lhs {
		positionGeneratedExpression(expression, position)
	}
	for _, expression := range statement.Rhs {
		positionGeneratedExpression(expression, position)
	}
}

func positionGeneratedDeclaration(declaration *ast.GenDecl, position token.Pos) {
	if declaration.TokPos == token.NoPos {
		declaration.TokPos = position
	}
	for _, specification := range declaration.Specs {
		value, ok := specification.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range value.Names {
			positionGeneratedExpression(name, position)
		}
		for _, expression := range value.Values {
			positionGeneratedExpression(expression, position)
		}
	}
}

func positionGeneratedIf(branch *ast.IfStmt, position token.Pos) {
	if branch.If == token.NoPos {
		branch.If = position
	}
	positionGeneratedExpression(branch.Cond, position)
	if branch.Body.Lbrace == token.NoPos {
		branch.Body.Lbrace = position
	}
	if branch.Body.Rbrace == token.NoPos {
		branch.Body.Rbrace = position
	}
	for _, statement := range branch.Body.List {
		positionGeneratedBranchStatement(statement, position)
	}
}

func positionGeneratedBranchStatement(statement ast.Stmt, position token.Pos) {
	if returned, ok := statement.(*ast.ReturnStmt); ok {
		if returned.Return == token.NoPos {
			returned.Return = position
		}
		for _, result := range returned.Results {
			positionGeneratedExpression(result, position)
		}
		return
	}
	positionGeneratedStatement(statement, position)
}

func positionGeneratedExpression(expression ast.Expr, position token.Pos) {
	switch node := expression.(type) {
	case *ast.Ident:
		if node.NamePos == token.NoPos {
			node.NamePos = position
		}
	case *ast.BasicLit:
		if node.ValuePos == token.NoPos {
			node.ValuePos = position
		}
	case *ast.BinaryExpr:
		if node.OpPos == token.NoPos {
			node.OpPos = position
		}
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Y, position)
	case *ast.CallExpr:
		if node.Lparen == token.NoPos {
			node.Lparen = position
		}
		if node.Rparen == token.NoPos {
			node.Rparen = position
		}
		positionGeneratedExpression(node.Fun, position)
		for _, argument := range node.Args {
			positionGeneratedExpression(argument, position)
		}
	case *ast.SelectorExpr:
		positionGeneratedExpression(node.X, position)
		positionGeneratedExpression(node.Sel, position)
	case *ast.CompositeLit:
		if node.Lbrace == token.NoPos {
			node.Lbrace = position
		}
		if node.Rbrace == token.NoPos {
			node.Rbrace = position
		}
	}
}

// directCall returns the call assignment and its error branch for a direct marker.
func (l *propagationLowerer) directCall(
	expression ast.Expr,
	targets int,
	context string,
) (*ast.AssignStmt, []ast.Stmt, int, bool) {
	metadata, direct := propagationMarker(l.source, expression)
	if !direct {
		return nil, nil, -1, false
	}
	values, statements := l.propagation(expression)
	if len(values) != targets {
		l.reportResultCount(metadata, len(values), targets, context)
		return nil, statements, -1, true
	}
	callIndex := len(statements) - 2
	if callIndex < 0 {
		return nil, statements, -1, true
	}
	callAssignment, ok := statements[callIndex].(*ast.AssignStmt)
	if !ok || len(callAssignment.Lhs) != targets+1 {
		return nil, statements, -1, true
	}
	for _, value := range values {
		if name, ok := value.(*ast.Ident); ok {
			delete(l.names, name.Name)
		}
	}
	return callAssignment, statements, callIndex, true
}

func identifierTargets(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		if _, ok := expression.(*ast.Ident); !ok {
			return false
		}
	}
	return true
}
