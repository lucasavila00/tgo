package compiler

import (
	"fmt"
	"go/scanner"
	"go/token"
)

// lex scans Go tokens and keeps their byte spans in the source.
func lex(name, input string) ([]lexeme, error) {
	fs := token.NewFileSet()
	file := fs.AddFile(name, -1, len(input))
	var scan scanner.Scanner
	var failure error
	scan.Init(file, []byte(input), func(pos token.Position, msg string) {
		if failure == nil {
			failure = fmt.Errorf("%s: %s", pos, msg)
		}
	}, scanner.ScanComments)
	result := []lexeme{}
	for {
		pos, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind == token.COMMENT {
			continue
		}
		position := file.Position(pos)
		start := file.Offset(pos)
		text := literal
		if text == "" {
			text = kind.String()
		}
		end := start + len(text)
		if kind == token.SEMICOLON && literal == "\n" {
			end = start
		}
		result = append(result, lexeme{
			kind:   kind,
			text:   text,
			start:  start,
			end:    end,
			line:   position.Line,
			column: position.Column,
		})
	}
	return result, failure
}

// closeToken finds the matching close token for one nested group.
func closeToken(tokens []lexeme, at int) (int, error) {
	stack := []token.Token{}
	for i := at; i < len(tokens); i++ {
		switch tokens[i].kind {
		case token.LPAREN:
			stack = append(stack, token.RPAREN)
		case token.LBRACK:
			stack = append(stack, token.RBRACK)
		case token.LBRACE:
			stack = append(stack, token.RBRACE)
		case token.RPAREN, token.RBRACK, token.RBRACE:
			if len(stack) == 0 || stack[len(stack)-1] != tokens[i].kind {
				return 0, fmt.Errorf(
					"%d:%d: unmatched %s",
					tokens[i].line,
					tokens[i].column,
					tokens[i].text,
				)
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf(
		"%d:%d: unclosed %s",
		tokens[at].line,
		tokens[at].column,
		tokens[at].text,
	)
}
