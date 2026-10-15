package scanner

import "go/token"

// Lexeme stores one scanned token and its exact source span.
type Lexeme struct {
	Pos     token.Pos
	Token   token.Token
	Literal string
	End     token.Pos
}

// Scan returns the next token and keeps its end separate from lookahead state.
func (s *Scanner) Scan() (token.Pos, token.Token, string) {
	var current Lexeme
	if len(s.pending) != 0 {
		current, s.pending = s.pending[0], s.pending[1:]
	} else {
		current = s.readLexeme()
	}
	s.deliveredEnd = current.End
	return current.Pos, current.Token, current.Literal
}

// End returns the end of the last token returned by Scan.
func (s *Scanner) End() token.Pos {
	return s.deliveredEnd
}

// Peek returns a future token without changing the token delivered to the parser.
// Each token is scanned once; Scan later consumes the same stored result.
func (s *Scanner) Peek(index int) Lexeme {
	for len(s.pending) <= index {
		s.pending = append(s.pending, s.readLexeme())
	}
	return s.pending[index]
}

// readLexeme captures a token's end before another scan changes scanner state.
func (s *Scanner) readLexeme() Lexeme {
	position, kind, literal := s.scan()
	return Lexeme{Pos: position, Token: kind, Literal: literal, End: s.scanEnd()}
}
