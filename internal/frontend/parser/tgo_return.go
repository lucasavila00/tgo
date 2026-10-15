package parser

import (
	"go/ast"
	"go/token"
)

// parseTGoReturnStmt reads a return list with an optional first or last ... element.
func (p *parser) parseTGoReturnStmt() *ast.ReturnStmt {
	statement := &ast.ReturnStmt{Return: p.expect(token.RETURN)}
	operator := token.NoPos
	zeroSuccess := p.tok == token.ELLIPSIS
	implicitEnd := false
	if zeroSuccess {
		operator = p.pos
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
				operator = p.pos
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
	if operator.IsValid() {
		p.recordShortReturn(statement, operator, zeroSuccess)
	}
	if !implicitEnd {
		p.expectSemi()
	}
	return statement
}

// recordShortReturn stores operands in one helper call at their original positions.
func (p *parser) recordShortReturn(statement *ast.ReturnStmt, operator token.Pos, zeroSuccess bool) {
	position, closing := operator, operator+2
	if len(statement.Results) > 0 {
		if statement.Results[0].Pos() < position {
			position = statement.Results[0].Pos()
		}
		if end := p.sourceExpressionEnd(statement.Results[len(statement.Results)-1]) - 1; end > closing {
			closing = end
		}
	}
	call := syntaxCall(position, closing, statement.Results)
	statement.Results = []ast.Expr{call}
	if p.forms != nil {
		p.forms.ShortReturns = append(p.forms.ShortReturns,
			ShortReturn{Statement: statement, Call: call, Position: operator, ZeroSuccess: zeroSuccess})
	}
}
