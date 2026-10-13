package scanner

import (
	"go/token"
	"reflect"
	"testing"
)

type scannedToken struct {
	kind    token.Token
	literal string
	offset  int
}

// scanInput returns the complete token sequence, including inserted semicolons.
func scanInput(input string, mode Mode) []scannedToken {
	set := token.NewFileSet()
	file := set.AddFile("main.tgo", -1, len(input))
	var scanner Scanner
	scanner.Init(file, []byte(input), nil, mode)
	var tokens []scannedToken
	for {
		position, kind, literal := scanner.Scan()
		tokens = append(tokens, scannedToken{kind, literal, file.Offset(position)})
		if kind == token.EOF {
			return tokens
		}
	}
}

// TestPostfixTokens checks suffix width and semicolon positions at newline, comment, and end of input.
func TestPostfixTokens(t *testing.T) {
	for _, input := range []string{"f()!", "f()!!", "f()!\n", "f()!!/*\ncomment*/"} {
		t.Run(input, func(t *testing.T) {
			tokens := scanInput(input, TGo|ScanComments)
			operator := tokens[3]
			if operator.kind != token.NOT || operator.offset != 3 ||
				(operator.literal != "!" && operator.literal != "!!") {
				t.Fatalf("operator: %+v", operator)
			}
			semicolon := tokens[len(tokens)-2]
			if semicolon.kind != token.SEMICOLON || semicolon.literal != "\n" ||
				semicolon.offset < 3+len(operator.literal) {
				t.Fatalf("semicolon: %+v", semicolon)
			}
		})
	}
}

// TestUnaryTokens checks that TGo mode retains ordinary Go tokens and semicolon insertion.
func TestUnaryTokens(t *testing.T) {
	for _, input := range []string{
		"return !\nfalse", "!!flag", "f() != nil", "f()\n!flag", "f()/*\n*/!flag",
		"//line virtual.go:20\nreturn !flag", "f()\n!\nflag",
	} {
		t.Run(input, func(t *testing.T) {
			ordinary := scanInput(input, ScanComments)
			withTGo := scanInput(input, ScanComments|TGo)
			if !reflect.DeepEqual(ordinary, withTGo) {
				t.Fatalf("ordinary: %+v\nTGo: %+v", ordinary, withTGo)
			}
		})
	}
}
