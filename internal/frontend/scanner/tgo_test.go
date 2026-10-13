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

// TestLookaheadKeepsTokenSpans checks that buffered tokens retain their original sequence and end positions.
func TestLookaheadKeepsTokenSpans(t *testing.T) {
	for _, input := range []string{
		"\ufeffpackage p\n//line virtual.tgo:40:2\nfunc f() { call()!! /*\ncomment*/ return value ..\n}",
		"return .. // before\nerr", "call()! // after\nnext()", "call()!!/*\ncomment*/next()",
	} {
		t.Run(input, func(t *testing.T) {
			set := token.NewFileSet()
			file := set.AddFile("main.tgo", -1, len(input))
			var baseline Scanner
			baseline.Init(file, []byte(input), nil, TGo|ScanComments)
			var expected []Lexeme
			for {
				position, kind, literal := baseline.scan()
				expected = append(expected, Lexeme{position, kind, literal, baseline.scanEnd()})
				if kind == token.EOF {
					break
				}
			}
			checkBufferedTokens(t, input, expected)
		})
	}
}

// checkBufferedTokens checks full lookahead without changing the last delivered token end.
func checkBufferedTokens(t *testing.T, input string, expected []Lexeme) {
	t.Helper()
	set := token.NewFileSet()
	file := set.AddFile("main.tgo", -1, len(input))
	var scanner Scanner
	scanner.Init(file, []byte(input), nil, TGo|ScanComments)
	for index, want := range expected {
		end := scanner.End()
		future := scanner.Peek(len(expected) - index - 1)
		if future != expected[len(expected)-1] || scanner.End() != end {
			t.Fatalf("lookahead: %+v; delivered end: %v, want %v", future, scanner.End(), end)
		}
		position, kind, literal := scanner.Scan()
		if got := (Lexeme{position, kind, literal, scanner.End()}); got != want {
			t.Fatalf("token %d: %+v, want %+v", index, got, want)
		}
	}
}

// TestScannerReuseDropsLookahead checks that initialization clears buffered tokens and delivered positions.
func TestScannerReuseDropsLookahead(t *testing.T) {
	set := token.NewFileSet()
	oldInput := "old()!!\n"
	oldFile := set.AddFile("old.tgo", -1, len(oldInput))
	var scanner Scanner
	scanner.Init(oldFile, []byte(oldInput), nil, TGo)
	scanner.Scan()
	scanner.Peek(3)
	input := "new()!\n"
	file := set.AddFile("new.tgo", -1, len(input))
	scanner.Init(file, []byte(input), nil, TGo)
	if scanner.End() != token.NoPos {
		t.Fatalf("old delivered end remains: %v", scanner.End())
	}
	position, kind, literal := scanner.Scan()
	if position != file.Pos(0) || kind != token.IDENT || literal != "new" || scanner.End() != file.Pos(3) {
		t.Fatalf("first token: %v %v %q; end: %v", position, kind, literal, scanner.End())
	}
}
