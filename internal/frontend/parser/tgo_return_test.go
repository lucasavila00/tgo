package parser

import (
	"go/ast"
	stock "go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// TestShortReturnGrammar checks operands, statement endings, and exact ellipsis positions.
func TestShortReturnGrammar(t *testing.T) {
	for _, body := range []string{
		"return ..., err",
		"return ...,\n err",
		"return ... /* before comma */, // before operand\n err // after\n",
		"return value, ...",
		"return x, y, ...",
		"return pair(), ...",
		"return value, ...\n next()",
		"return value, ... /* after\n */ next()",
		"return call()!, ...",
		"return ..., call()!!",
		"return ..., first() + call()!",
		"return if ready { return ..., err; 1 } else { 2 }, ...",
		"return []int{if ready { return ..., err; x } else { x } for x := range 2}, ...",
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
	if position.Filename != "main.tgo" || input[position.Offset:position.Offset+3] != "..." {
		t.Fatalf("ellipsis position: %v", position)
	}
	if len(form.Statement.Results) != 1 || form.Statement.Results[0] != form.Call {
		t.Fatal("return record does not identify the parsed node")
	}
	if name := form.Call.Fun.(*ast.Ident).Name; name != "" {
		t.Fatalf("parser assigned a helper name: %q", name)
	}
	if form.Statement.End() < form.Position+3 {
		t.Fatal("ellipsis outside return source span")
	}
	for _, operand := range form.Call.Args {
		if operand.Pos() < form.Statement.Pos() || operand.End() > form.Statement.End() {
			t.Fatal("operand outside source return span")
		}
	}
}

// TestShortReturnGrammarRejectsWrongPositions checks missing commas and misplaced ellipses.
func TestShortReturnGrammarRejectsWrongPositions(t *testing.T) {
	for _, body := range []string{
		"return ...", "return ...,", "return ... err", "return value ...",
		"return ..., err, ...", "return x, ..., y", "return ..., ..., err", "return ( ... ), err",
		"return xs..., ...", "return ..., xs...", "return x, ... y",
		"return .. err", "return value ..", "return call(values..)", "return 1..", "return []..int{}",
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
	input := "package p; func f() (int, error) { return func() int { _ = x ..; return 1 }(), ... }"
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
		"return value, ...\nnext()",
		"return value, ... /* after\n */ next()",
		"return value, ... // after\nnext()",
		"return value, ...; next()",
		"return ...,\nerr\nnext()",
		"return value, ...\n//line virtual.tgo:1\nnext()",
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
	input := "package p; func f() { return value, ... /*line virtual.tgo:99:1*/ next() }"
	_, _, err := ParseTGoFile(token.NewFileSet(), "main.go", "main.tgo", input,
		ParseComments|AllErrors|SkipObjectResolution)
	if err == nil || !strings.Contains(err.Error(), "short return requires ... to be the first or last result") {
		t.Fatalf("same-line diagnostic: %v", err)
	}
}

// TestOrdinaryEllipsesMatchGo checks variadic calls, variadic types, and inferred array lengths.
func TestOrdinaryEllipsesMatchGo(t *testing.T) {
	for _, declaration := range []string{
		"func f(xs ...int) int { return len(xs) }",
		"func f(xs []int) int { return sum(xs...) }",
		"func f() [2]int { return [...]int{1, 2} }",
		"func f(xs []int) (int, error) { return sum(xs...), nil }",
		"func f(xs []int) func(...int) { return func(ys ...int) {} }",
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
