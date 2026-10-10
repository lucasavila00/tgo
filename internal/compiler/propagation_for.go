package compiler

import (
	"go/ast"
	"go/token"
)

func (l *propagationLowerer) forStatement(node *ast.ForStmt) []ast.Stmt {
	outer := l.inferredResultNames
	l.inferredResultNames = cloneInferredResultNames(outer)
	scopedInitializer := node.Init != nil
	prefix := []ast.Stmt(nil)
	if l.statementHasLowering(node.Init) {
		if assignment, ok := node.Init.(*ast.AssignStmt); ok {
			l.rememberSimpleAssignmentTypes(assignment)
			l.rememberDirectResultTypes(
				assignment.Lhs,
				l.directPropagationSignature(assignment.Rhs[0]),
			)
			lowered := l.lowerAssignment(assignment, false)
			node.Init = lowered[len(lowered)-1]
			prefix = lowered[:len(lowered)-1]
		} else {
			prefix = l.simpleStatement(node.Init)
			node.Init = nil
		}
	} else if assignment, ok := node.Init.(*ast.AssignStmt); ok {
		l.rememberSimpleAssignmentTypes(assignment)
	}
	l.missingStatementLowering(node.Init, "for initializer")
	node.Body.List = l.scopedStatements(node.Body.List)
	postPrefix := []ast.Stmt(nil)
	if l.statementHasLowering(node.Post) {
		// Keep source post work after Go creates the next iteration variables.
		pending := l.freshName("post")
		prefix = append(prefix, &ast.AssignStmt{
			Lhs: []ast.Expr{pending}, Tok: token.DEFINE,
			Rhs: []ast.Expr{l.unit.generatedUniverse("false", node.Post.Pos())},
		})
		postPrefix = append(postPrefix, &ast.IfStmt{
			Cond: ast.NewIdent(pending.Name),
			Body: &ast.BlockStmt{List: append([]ast.Stmt{&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN,
				Rhs: []ast.Expr{l.unit.generatedUniverse("false", node.Post.Pos())},
			}}, l.simpleStatement(node.Post)...)},
		})
		node.Post = &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{l.unit.generatedUniverse("true", node.Post.Pos())},
		}
	}
	if len(postPrefix) > 0 || l.hasLowering(node.Cond) {
		condition, conditionPrefix := l.expression(node.Cond)
		body := make(
			[]ast.Stmt, 0,
			len(postPrefix)+len(conditionPrefix)+1+len(node.Body.List),
		)
		body = append(body, postPrefix...)
		body = append(body, conditionPrefix...)
		if condition != nil {
			body = append(body, &ast.IfStmt{
				Cond: &ast.UnaryExpr{Op: token.NOT, X: condition},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{
					Tok: token.BREAK,
				}}},
			})
		}
		node.Cond = nil
		body = append(body, node.Body.List...)
		node.Body.List = body
	}
	result := l.prefixedStatement(prefix, node, scopedInitializer)
	l.inferredResultNames = outer
	return result
}
