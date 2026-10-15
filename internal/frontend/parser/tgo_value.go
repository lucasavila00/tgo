package parser

import (
	"go/ast"
	"go/token"
)

// parseValueConditional reads an if or switch with ordinary statement grammar.
// Its function literal stores syntax; it does not introduce a source function.
func (p *parser) parseValueConditional() ast.Expr {
	headerAssignment := p.headerAssignment
	p.headerAssignment = false
	p.exprLev++
	var statement ast.Stmt
	if p.tok == token.IF {
		statement = p.parseIf(true)
	} else {
		statement = p.parseSwitch(true)
	}
	p.exprLev--
	p.headerAssignment = headerAssignment
	position, closing := statement.Pos(), statement.End()-1
	body := &ast.BlockStmt{Lbrace: position, List: []ast.Stmt{statement}, Rbrace: closing}
	call := syntaxCall(position, closing, []ast.Expr{syntaxFunction(position, body)})
	if p.forms != nil {
		p.forms.Conditionals = append(p.forms.Conditionals, Conditional{Call: call, Position: position})
	}
	return call
}

// syntaxCall creates an unnamed helper at the source expression's boundaries.
func syntaxCall(position, closing token.Pos, arguments []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.Ident{NamePos: position}, Lparen: position, Args: arguments, Rparen: closing,
	}
}

// syntaxFunction stores statements for the checker without creating a source function scope.
func syntaxFunction(position token.Pos, body *ast.BlockStmt) *ast.FuncLit {
	return &ast.FuncLit{
		Type: &ast.FuncType{Func: position, Params: &ast.FieldList{Opening: position, Closing: position}},
		Body: body,
	}
}

// sourceExpressionEnd includes a final postfix suffix while the operand tree is unchanged.
func (p *parser) sourceExpressionEnd(expression ast.Expr) token.Pos {
	end := expression.End()
	if suffixEnd := p.postfixEnds[end]; suffixEnd > end {
		return suffixEnd
	}
	return end
}
