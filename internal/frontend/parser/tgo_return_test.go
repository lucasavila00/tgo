package parser

import (
	"go/ast"
	stock "go/parser"
	"go/token"
	"reflect"
	"strings"
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

// TestOrdinaryDotPairsMatchGo checks dot errors and recovery in each expression position.
func TestOrdinaryDotPairsMatchGo(t *testing.T) {
	for _, body := range []string{
		"call(..)", "_ = xs[..]", "_ = (..x)", "return (..x)",
		"call(x..)", "_ = xs[x..]", "_ = (x..)", "_ = x ..",
		"call(x..\n)", "_ = xs[x..\n]", "call(x../*\n*/)", "_ = x ..; next()",
		"return call(x..)", "_ = x.(..)", "_ = x.(..T)", "_ = x . .", "_ = x ./*gap*/.", "_ = x .\n.",
	} {
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

// TestShortReturnContextStopsAtFunction checks that a nested function owns its return grammar.
func TestShortReturnContextStopsAtFunction(t *testing.T) {
	input := "package p; func f() (int, error) { return func() int { _ = x ..; return 1 }() .. }"
	_, _, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input,
		AllErrors|SkipObjectResolution)
	if err == nil || !strings.Contains(err.Error(), "expected selector or type assertion") {
		t.Fatalf("nested function diagnostic: %v", err)
	}
}

// TestDotTypeGrammarMatchesGo keeps Go's partial syntax trees and errors in type positions.
func TestDotTypeGrammarMatchesGo(t *testing.T) {
	for _, declaration := range []string{
		"var x ..int",
		"type T struct{ Field ..int }",
		"func f[T ..](){}",
		"type T ..int",
		"func f(v ..int){}",
		"var _ = []..int{}",
		"type S struct { value ..T }",
		"type S struct { ..T }",
		"func f(value ..T) {}",
		"func f(..T) {}",
		"func f() (..T) {}",
		"type G[T ..int] struct{}",
		"type C interface { ~..T }",
		"type Alias = pkg..T",
		"type S struct { pkg..T }",
		"var channel chan ..T",
		"var array [..]T",
	} {
		t.Run(declaration, func(t *testing.T) {
			input := "package p; " + declaration
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

// TestShortReturnKeepsNextStatement checks that a line ending leaves the next statement for its body.
func TestShortReturnKeepsNextStatement(t *testing.T) {
	for _, body := range []string{
		"return value ..\nnext()",
		"return value .. /* after\n */ next()",
		"return value .. // after\nnext()",
		"return value ..; next()",
		"return ..\nerr\nnext()",
		"return value ..\n//line virtual.tgo:1\nnext()",
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p\nfunc f() { " + body + "\n}"
			file, forms, err := ParseTGoFile(token.NewFileSet(), "main.go", "main.tgo", input,
				ParseComments|AllErrors|SkipObjectResolution)
			if err != nil || len(forms.ShortReturns) != 1 {
				t.Fatalf("return records: %d; error: %v", len(forms.ShortReturns), err)
			}
			statements := file.Decls[0].(*ast.FuncDecl).Body.List
			if len(statements) != 2 {
				t.Fatalf("body contains %d statements; want the return and next()", len(statements))
			}
			call := statements[1].(*ast.ExprStmt).X.(*ast.CallExpr)
			if call.Fun.(*ast.Ident).Name != "next" {
				t.Fatal("the next statement changed")
			}
		})
	}
}

// TestShortReturnUsesPhysicalLine checks that a line directive cannot end a statement on the same source line.
func TestShortReturnUsesPhysicalLine(t *testing.T) {
	input := "package p; func f() { return value .. /*line virtual.tgo:99:1*/ next() }"
	_, _, err := ParseTGoFile(token.NewFileSet(), "main.go", "main.tgo", input,
		ParseComments|AllErrors|SkipObjectResolution)
	if err == nil || !strings.Contains(err.Error(), "short return permits .. only before or after its operands") {
		t.Fatalf("same-line diagnostic: %v", err)
	}
}
