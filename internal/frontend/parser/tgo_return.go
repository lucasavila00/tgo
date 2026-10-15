package parser

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
)

// parseTGoReturnStmt reads return operands and a first or last ellipsis placeholder.
func (p *parser) parseTGoReturnStmt() ast.Stmt {
	statement := &ast.ReturnStmt{Return: p.expect(token.RETURN)}
	operator, operatorEnd := token.NoPos, token.NoPos
	zeroSuccess := p.tok == token.ELLIPSIS
	implicitEnd := false
	if zeroSuccess {
		operator, operatorEnd = p.pos, p.end()
		p.next()
		p.expect(token.COMMA)
		if p.tok == token.SEMICOLON || p.tok == token.RBRACE || p.tok == token.EOF {
			p.error(p.pos, "short return requires an error operand")
		} else {
			statement.Results = p.parseList(true)
		}
	} else if p.tok != token.SEMICOLON && p.tok != token.RBRACE {
		statement.Results = append(statement.Results, p.parseRhs())
		for p.tok == token.COMMA {
			p.next()
			if p.tok == token.ELLIPSIS {
				operator, operatorEnd = p.pos, p.end()
				line := p.lineFor(p.pos)
				p.next()
				implicitEnd = p.tok == token.EOF || p.lineFor(p.pos) > line
				if !implicitEnd && p.tok != token.SEMICOLON && p.tok != token.RBRACE {
					p.error(operator, "short return requires ... to be the first or last result")
				}
				break
			}
			statement.Results = append(statement.Results, p.parseRhs())
		}
	}
	if !implicitEnd {
		p.expectSemi()
	}
	if operator.IsValid() {
		return &ast.ShortReturnStmt{Return: statement.Return, Placeholder: operator,
			PlaceholderEnd: operatorEnd, Results: statement.Results, ZeroSuccess: zeroSuccess}
	}
	return statement
}
