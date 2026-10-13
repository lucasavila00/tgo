package parser

import (
	"go/ast"
	"go/token"
)

// Postfix constructs a checker call for an operand and its postfix operator.
// The operator span uses positions in the parsed file; end is exclusive.
type Postfix func(operand ast.Expr, start, end token.Pos, wrap bool) ast.Expr

// parsePostfix consumes one operator on the current operand and creates its checker call.
func (p *parser) parsePostfix(operand ast.Expr) ast.Expr {
	start, end, wrap := p.pos, p.end(), p.lit == "!!"
	p.next()
	if _, call := ast.Unparen(operand).(*ast.CallExpr); !call {
		p.error(start, "propagation requires a call")
		return &ast.BadExpr{From: operand.Pos(), To: end}
	}
	if p.postfix != nil {
		return p.postfix(operand, start, end, wrap)
	}
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "tgoPropagation", NamePos: operand.Pos()},
		Lparen: operand.Pos(), Args: []ast.Expr{ast.Unparen(operand)}, Rparen: end - 1,
	}
}
