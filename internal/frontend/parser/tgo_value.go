package parser

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
)

// parseValueConditional reads branches in the enclosing source expression context.
func (p *parser) parseValueConditional() ast.Expr {
	headerAssignment := p.headerAssignment
	p.headerAssignment = false
	p.exprLev++
	var expression ast.Expr
	if p.tok == token.IF {
		expression = valueIf(p.parseIf(true))
	} else {
		expression = &ast.SwitchExpr{Switch: p.parseSwitch(true)}
	}
	p.exprLev--
	p.headerAssignment = headerAssignment
	return expression
}

// valueIf retains the parsed header and branches in native expression nodes.
func valueIf(statement *ast.IfStmt) *ast.IfExpr {
	var alternative ast.Node = statement.Else
	if nested, ok := statement.Else.(*ast.IfStmt); ok {
		alternative = valueIf(nested)
	}
	return &ast.IfExpr{If: statement.If, Init: statement.Init, Cond: statement.Cond,
		Body: statement.Body, Else: alternative}
}
