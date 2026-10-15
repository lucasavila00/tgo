package parser

import "github.com/lucasavila00/tgo/internal/ast"

// parsePostfix reads one source operator and retains its immediate operand.
func (p *parser) parsePostfix(operand ast.Expr) ast.Expr {
	start, end, wrap := p.pos, p.end(), p.lit == "!!"
	p.next()
	call := ast.Unparen(operand)
	for {
		propagation, nested := call.(*ast.PropagateExpr)
		if !nested {
			break
		}
		call = ast.Unparen(propagation.X)
	}
	if _, isCall := call.(*ast.CallExpr); !isCall {
		p.error(start, "propagation requires a call")
		return &ast.BadExpr{From: operand.Pos(), To: end}
	}
	return &ast.PropagateExpr{X: operand, OpPos: start, OpEnd: end, Wrap: wrap}
}
