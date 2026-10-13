package parser

import (
	"go/ast"
	"go/token"
)

// Postfix records the exact operand and its postfix operator.
// The operator span uses positions in the parsed file; end is exclusive.
type Postfix func(operand ast.Expr, start, end token.Pos, wrap bool)

// parsePostfix consumes one operator and records its current call operand.
func (p *parser) parsePostfix(operand ast.Expr) ast.Expr {
	start, end, wrap := p.pos, p.end(), p.lit == "!!"
	p.next()
	if _, call := ast.Unparen(operand).(*ast.CallExpr); !call {
		p.error(start, "propagation requires a call")
		return &ast.BadExpr{From: operand.Pos(), To: end}
	}
	if p.postfix != nil {
		p.postfix(operand, start, end, wrap)
	}
	return operand
}
