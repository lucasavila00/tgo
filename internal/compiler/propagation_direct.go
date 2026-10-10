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
	value.Names = append(value.Names, errorName)
	value.Values = call.Rhs
	statements[callIndex] = node
	if value.Comment != nil {
		for _, statement := range statements[callIndex+1:] {
			positionGeneratedErrorBranch(statement, value.Comment.End()+1)
		}
	}
	return statements, true
}

func positionGeneratedErrorBranch(statement ast.Stmt, position token.Pos) {
	branch, ok := statement.(*ast.IfStmt)
	if !ok || branch.If != token.NoPos {
		return
	}
	branch.If = position
	if condition, ok := branch.Cond.(*ast.BinaryExpr); ok {
		condition.OpPos = position
		positionGeneratedExpression(condition.X, position)
		positionGeneratedExpression(condition.Y, position)
	}
	branch.Body.Lbrace = position
	branch.Body.Rbrace = position
	for _, statement := range branch.Body.List {
		switch node := statement.(type) {
		case *ast.ReturnStmt:
			node.Return = position
			for _, result := range node.Results {
				positionGeneratedExpression(result, position)
			}
		case *ast.DeclStmt:
			if declaration, ok := node.Decl.(*ast.GenDecl); ok {
				declaration.TokPos = position
				for _, specification := range declaration.Specs {
					if value, ok := specification.(*ast.ValueSpec); ok {
						for _, name := range value.Names {
							name.NamePos = position
						}
					}
				}
			}
		}
	}
}

func positionGeneratedExpression(expression ast.Expr, position token.Pos) {
	switch node := expression.(type) {
	case *ast.Ident:
		node.NamePos = position
	case *ast.BasicLit:
		node.ValuePos = position
	case *ast.CallExpr:
		node.Lparen = position
		node.Rparen = position
		positionGeneratedExpression(node.Fun, position)
		for _, argument := range node.Args {
			positionGeneratedExpression(argument, position)
		}
	case *ast.SelectorExpr:
		positionGeneratedExpression(node.X, position)
		node.Sel.NamePos = position
	case *ast.CompositeLit:
		node.Lbrace = position
		node.Rbrace = position
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
