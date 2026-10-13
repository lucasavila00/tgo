package frontend

import (
	"github.com/lucasavila00/tgo/internal/frontend/scanner"
	"go/token"
	"strings"
)

type sourceToken struct {
	offset int
	end    int
	kind   token.Token
	depth  int
}

// sourceTokens scans text into tokens with byte spans and delimiter depth.
// Syntax planners use these records to find expression and statement boundaries.
func sourceTokens(source string) []sourceToken {
	set := token.NewFileSet()
	file := set.AddFile("", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, []byte(source), nil, scanner.TGo)
	var tokens []sourceToken
	depth := 0
	for {
		position, kind, literal := scan.Scan()
		width := len(literal)
		if width == 0 {
			width = len(kind.String())
		}
		offset := file.Offset(position)
		if kind == token.STRING && source[offset] == '`' {
			width = strings.IndexByte(source[offset+1:], '`') + 2
		}
		tokens = append(tokens, sourceToken{offset: offset, end: offset + width, kind: kind, depth: depth})
		switch kind {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
		if kind == token.EOF {
			return tokens
		}
	}
}
