package compiler

import (
	"go/ast"
	"go/token"
	"strconv"
)

func (l *propagationLowerer) caseBodies(body *ast.BlockStmt) {
	for _, item := range body.List {
		clause := item.(*ast.CaseClause)
		clause.Body = l.scopedStatements(clause.Body)
	}
}

func (l *propagationLowerer) diagnoseSwitchCases(body *ast.BlockStmt) {
	for _, item := range body.List {
		clause := item.(*ast.CaseClause)
		for _, expression := range clause.List {
			l.missingExpressionLowering(expression, "switch case")
		}
	}
}

func (l *propagationLowerer) switchCasesHavePropagation(body *ast.BlockStmt) bool {
	for _, item := range body.List {
		clause := item.(*ast.CaseClause)
		for _, expression := range clause.List {
			if l.statementHasLowering(&ast.ExprStmt{X: expression}) {
				return true
			}
		}
	}
	return false
}

// lowerSwitchCases selects one source clause before the switch runs its body.
func (l *propagationLowerer) lowerSwitchCases(
	body *ast.BlockStmt,
	tag ast.Expr,
) (ast.Expr, []ast.Stmt) {
	statements := []ast.Stmt(nil)
	var savedTag *ast.Ident
	if tag != nil {
		name := l.freshName("tag")
		statements = append(statements, &ast.AssignStmt{
			Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{tag},
		})
		savedTag = ast.NewIdent(name.Name)
	}

	selected := l.freshName("selected")
	statements = append(statements, &ast.AssignStmt{
		Lhs: []ast.Expr{selected}, Tok: token.DEFINE,
		Rhs: []ast.Expr{switchNegativeOne()},
	})
	for clauseIndex, item := range body.List {
		clause := item.(*ast.CaseClause)
		if len(clause.List) == 0 {
			continue
		}
		for _, sourceExpression := range clause.List {
			expression, prefix := l.expression(sourceExpression)
			condition := expression
			if savedTag != nil {
				condition = &ast.BinaryExpr{
					X:  ast.NewIdent(savedTag.Name),
					Op: token.EQL,
					Y:  expression,
				}
				positionGeneratedExpression(condition, sourceExpression.Pos())
			}
			prefix = append(prefix, &ast.IfStmt{
				Cond: condition,
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent(selected.Name)}, Tok: token.ASSIGN,
					Rhs: []ast.Expr{switchIntegerLiteral(clauseIndex)},
				}}},
			})
			statements = append(statements, &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X: ast.NewIdent(selected.Name), Op: token.EQL, Y: switchNegativeOne(),
				},
				Body: &ast.BlockStmt{List: prefix},
			})
		}
		clause.List = []ast.Expr{switchIntegerLiteral(clauseIndex)}
	}
	return ast.NewIdent(selected.Name), statements
}

func switchNegativeOne() ast.Expr {
	return &ast.UnaryExpr{Op: token.SUB, X: switchIntegerLiteral(1)}
}

func switchIntegerLiteral(value int) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)}
}
