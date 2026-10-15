package parser

import (
	"go/ast"
	"go/token"
)

// parsePostfix consumes one operator and records its current call operand.
func (p *parser) parsePostfix(operand ast.Expr) ast.Expr {
	start, end, wrap := p.pos, p.end(), p.lit == "!!"
	p.next()
	if _, call := ast.Unparen(operand).(*ast.CallExpr); !call {
		p.error(start, "propagation requires a call")
		return &ast.BadExpr{From: operand.Pos(), To: end}
	}
	if p.postfixEnds == nil {
		p.postfixEnds = make(map[token.Pos]token.Pos)
	}
	p.postfixEnds[operand.End()] = end
	if p.forms != nil {
		p.forms.Postfix = append(p.forms.Postfix, Postfix{Operand: operand, Start: start, End: end, Wrap: wrap})
	}
	return operand
}
