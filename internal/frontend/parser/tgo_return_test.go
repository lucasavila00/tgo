package parser

import (
	"go/ast"
	stock "go/parser"
	"go/token"
	"reflect"
	"testing"
)

// TestShortReturnGrammar checks operands, statement endings, and exact operator positions.
func TestShortReturnGrammar(t *testing.T) {
	for _, body := range []string{
		"return .. err",
		"return ..\n err",
		"return .. // before\n err // after\n",
		"return value ..",
		"return x, y ..",
		"return pair() ..",
		"return value ..\n next()",
		"return value .. /* after\n */ next()",
		"return call()! ..",
		"return .. call()!!",
		"return .. first() + call()!",
		"return if ready { return .. err; 1 } else { 2 } ..",
		"return []int{if ready { return .. err; x } else { x } for x := range 2} ..",
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p\nfunc f() (int,error) { " + body + "\n}"
			positions := token.NewFileSet()
			_, forms, err := ParseTGoFile(positions, "main_tgo.go", "main.tgo", input,
				ParseComments|AllErrors|SkipObjectResolution)
			if err != nil || len(forms.ShortReturns) == 0 {
				t.Fatalf("returns: %d; error: %v", len(forms.ShortReturns), err)
			}
			for _, form := range forms.ShortReturns {
				checkShortReturnContainer(t, positions, input, form)
			}
		})
	}
}

// checkShortReturnContainer checks the recorded return node and operand source spans.
func checkShortReturnContainer(t *testing.T, positions *token.FileSet, input string, form ShortReturn) {
	t.Helper()
	position := positions.Position(form.Position)
	if position.Filename != "main.tgo" || input[position.Offset:position.Offset+2] != ".." {
		t.Fatalf("operator position: %v", position)
	}
	if len(form.Statement.Results) != 1 || form.Statement.Results[0] != form.Call {
		t.Fatal("return record does not identify the parsed node")
	}
	if name := form.Call.Fun.(*ast.Ident).Name; name != "" {
		t.Fatalf("parser assigned a helper name: %q", name)
	}
	for _, operand := range form.Call.Args {
		if operand.Pos() < form.Statement.Pos() || operand.End() > form.Statement.End() {
			t.Fatal("operand outside source return span")
		}
	}
}

// TestShortReturnGrammarRejectsWrongPositions checks .. outside the two return forms.
func TestShortReturnGrammarRejectsWrongPositions(t *testing.T) {
	for _, body := range []string{
		"return ..", "return .. err ..", "return x .. y", "return x .., y", "return .., err",
		"return call(values..)", "return 1..", "return []..int{}",
	} {
		t.Run(body, func(t *testing.T) {
			_, _, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo",
				"package p; func f() { "+body+" }", AllErrors|SkipObjectResolution)
			if err == nil {
				t.Fatal("invalid short return parsed")
			}
		})
	}
}

// TestShortReturnEllipsisRecord keeps the ellipsis for the checker to reject with the source position.
func TestShortReturnEllipsisRecord(t *testing.T) {
	for _, body := range []string{
		"return xs... ..", "return xs... /* comment */ ..", "return .. xs...", "return xs... ..\nnext()",
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p; func f() { " + body + " }"
			positions := token.NewFileSet()
			_, forms, err := ParseTGoFile(positions, "main.tgo", "main.tgo", input, AllErrors|SkipObjectResolution)
			if err != nil || len(forms.ShortReturns) != 1 {
				t.Fatalf("return records: %d; error: %v", len(forms.ShortReturns), err)
			}
			call := forms.ShortReturns[0].Call
			if !call.Ellipsis.IsValid() {
				t.Fatal("missing ellipsis position")
			}
			offset := positions.Position(call.Ellipsis).Offset
			if input[offset:offset+3] != "..." || call.End() < call.Ellipsis+3 {
				t.Fatalf("ellipsis position or call end changed: %+v", call)
			}
		})
	}
}

// TestOrdinaryReturnEllipsisMatchesGo keeps Go's diagnostic for an ordinary invalid return.
func TestOrdinaryReturnEllipsisMatchesGo(t *testing.T) {
	for _, body := range []string{"return xs...", "return xs... // comment\n return nil"} {
		t.Run(body, func(t *testing.T) {
			input := "package p; func f() { " + body + " }"
			mode := ParseComments | AllErrors | SkipObjectResolution
			got, gotError := ParseFile(token.NewFileSet(), "main.go", input, mode)
			want, wantError := stock.ParseFile(token.NewFileSet(), "main.go", input, stock.Mode(mode))
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotError, wantError) {
				t.Fatalf("AST differs: %t; diagnostics: %v, want %v", !reflect.DeepEqual(got, want),
					gotError, wantError)
			}
		})
	}
}
