package parser

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/frontend/scanner"
)

// parseTGoReturnStmt reads ordinary results or results shortened with .. .
// The scanner ends a trailing form at a newline, comment newline, or EOF.
func (p *parser) parseTGoReturnStmt() *ast.ReturnStmt {
	position := p.expect(token.RETURN)
	statement := &ast.ReturnStmt{Return: position}
	operator := token.NoPos
	ellipsis, ellipsisEnd := token.NoPos, token.NoPos
	zeroSuccess := p.tok == scanner.ShortReturn
	if zeroSuccess {
		operator = p.pos
		p.next()
	}
	if p.tok != token.SEMICOLON && p.tok != token.RBRACE && p.tok != token.EOF {
		statement.Results = p.parseList(true)
	} else if zeroSuccess {
		p.error(operator, "short return requires an error operand")
	}
	if p.tok == token.ELLIPSIS {
		following, _ := p.peekHeaderToken(0)
		if zeroSuccess || following.Token == scanner.ShortReturn {
			ellipsis, ellipsisEnd = p.pos, p.end()
			p.next()
		}
	}
	if p.tok == scanner.ShortReturn {
		if zeroSuccess {
			p.error(p.pos, "short return permits .. only before or after its operands")
		}
		operator = p.pos
		p.next()
	}
	if operator.IsValid() {
		p.recordShortReturn(statement, operator, zeroSuccess, ellipsis, ellipsisEnd)
	}
	p.expectSemi()
	return statement
}

// recordShortReturn stores operands in one helper call at their original positions.
func (p *parser) recordShortReturn(statement *ast.ReturnStmt, operator token.Pos, zeroSuccess bool,
	ellipsis, ellipsisEnd token.Pos) {
	position, closing := operator, operator+1
	if len(statement.Results) > 0 {
		if statement.Results[0].Pos() < position {
			position = statement.Results[0].Pos()
		}
		if end := p.sourceExpressionEnd(statement.Results[len(statement.Results)-1]) - 1; end > closing {
			closing = end
		}
	}
	if ellipsisEnd-1 > closing {
		closing = ellipsisEnd - 1
	}
	call := syntaxCall(position, closing, statement.Results)
	call.Ellipsis = ellipsis
	statement.Results = []ast.Expr{call}
	if p.forms != nil {
		p.forms.ShortReturns = append(p.forms.ShortReturns,
			ShortReturn{Statement: statement, Call: call, Position: operator, ZeroSuccess: zeroSuccess})
	}
}
