package parser

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/frontend/scanner"
)

// startsHeaderComprehension distinguishes a named collection from a statement body.
// Tokens stay in the scanner buffer until the normal parser consumes them.
func (p *parser) startsHeaderComprehension() bool {
	braces, parentheses, brackets := 1, 0, 0
	index := 0
	generator := false
	for braces > 0 {
		current, next := p.peekHeaderToken(index)
		index = next
		if current.Token == token.EOF {
			return false
		}
		atCollection := braces == 1 && parentheses == 0 && brackets == 0
		if atCollection && !generator && current.Token == token.FOR {
			generator = true
		}
		switch current.Token {
		case token.LBRACE:
			braces++
		case token.RBRACE:
			braces--
		case token.LPAREN:
			parentheses++
		case token.RPAREN:
			parentheses--
		case token.LBRACK:
			brackets++
		case token.RBRACK:
			brackets--
		}
	}
	if !generator {
		return false
	}
	return p.headerExpressionFollows(index)
}

// headerExpressionFollows checks whether the token after the brace requires an expression.
func (p *parser) headerExpressionFollows(index int) bool {
	following, _ := p.peekHeaderToken(index)
	switch following.Token {
	case token.LBRACK, token.PERIOD, token.LPAREN, token.LBRACE:
		return true
	case token.SEMICOLON:
		return p.headerAssignment
	}
	return following.Token.Precedence() > 0
}

// peekHeaderToken skips comments while preserving each token in the shared buffer.
func (p *parser) peekHeaderToken(index int) (scanner.Lexeme, int) {
	for {
		current := p.scanner.Peek(index)
		index++
		if current.Token != token.COMMENT {
			return current, index
		}
	}
}
