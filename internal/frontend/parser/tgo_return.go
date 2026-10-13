package parser

import (
	"go/ast"
	"go/token"
)

// parseTGoReturnStmt reads return operands and an adjacent pair of dots.
// A trailing pair ends at the next physical line, explicit semicolon, closing brace, or EOF.
func (p *parser) parseTGoReturnStmt() *ast.ReturnStmt {
	previousLevel := p.returnOperandLevel
	p.returnOperandLevel = p.exprLev
	defer func() { p.returnOperandLevel = previousLevel }()
	position := p.expect(token.RETURN)
	statement := &ast.ReturnStmt{Return: position}
	operator := token.NoPos
	ellipsis, ellipsisEnd := token.NoPos, token.NoPos
	zeroSuccess := p.startsShortReturn()
	if zeroSuccess {
		operator = p.pos
		p.consumeShortReturn()
	}
	if p.tok != token.SEMICOLON && p.tok != token.RBRACE && p.tok != token.EOF {
		statement.Results = p.parseList(true)
	} else if zeroSuccess {
		p.error(operator, "short return requires an error operand")
	}
	if p.tok == token.ELLIPSIS && (zeroSuccess || p.shortReturnFollows()) {
		ellipsis, ellipsisEnd = p.pos, p.end()
		p.next()
	}
	implicitEnd := false
	if p.startsShortReturn() {
		if zeroSuccess {
			p.error(p.pos, "short return permits .. only before or after its operands")
		}
		operator = p.pos
		lastLine := p.consumeShortReturn()
		implicitEnd = p.tok == token.EOF || p.lineFor(p.pos) > lastLine
		if !implicitEnd && p.tok != token.SEMICOLON && p.tok != token.RBRACE {
			p.error(operator, "short return permits .. only before or after its operands")
		}
	}
	if operator.IsValid() {
		p.recordShortReturn(statement, operator, zeroSuccess, ellipsis, ellipsisEnd)
	}
	if !implicitEnd {
		p.expectSemi()
	}
	return statement
}

// startsShortReturn checks two adjacent Go period tokens at the current parser position.
func (p *parser) startsShortReturn() bool {
	if p.tok != token.PERIOD {
		return false
	}
	next := p.scanner.Peek(0)
	return next.Token == token.PERIOD && next.Pos == p.end()
}

// shortReturnFollows checks an adjacent pair immediately after the current ellipsis.
func (p *parser) shortReturnFollows() bool {
	first, next := p.peekHeaderToken(0)
	second := p.scanner.Peek(next)
	return first.Token == token.PERIOD && second.Token == token.PERIOD && second.Pos == first.End
}

// consumeShortReturn consumes both dots and returns the physical line of the second dot.
func (p *parser) consumeShortReturn() int {
	p.expect(token.PERIOD)
	line := p.lineFor(p.pos)
	p.expect(token.PERIOD)
	return line
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
