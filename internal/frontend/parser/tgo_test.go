package parser

import (
	"github.com/lucasavila00/tgo/internal/ast"
	stock "go/parser"
	"go/token"
	"reflect"
	"testing"
)

// TestOrdinaryGoSyntax checks AST and diagnostic compatibility for syntax without postfix operators.
func TestOrdinaryGoSyntax(t *testing.T) {
	for _, input := range []string{
		"//go:build linux\n\npackage p\nfunc f[T any](v T) T { return v }",
		"package p\n//line virtual.go:40\nfunc f() bool { return !\nfalse }",
		"package p\nfunc f() { for i := range 3 { _ = i }; _ = []string{`é`} }",
		"package p\nfunc f() { _ = call() != nil /* source */ }",
		"package p\nfunc f() { _ = }",
	} {
		t.Run(input, func(t *testing.T) {
			mode := ParseComments | AllErrors | SkipObjectResolution
			got, gotError := ParseFile(token.NewFileSet(), "main.go", input, mode)
			want, wantError := stock.ParseFile(token.NewFileSet(), "main.go", input, stock.Mode(mode))
			if !sameGoTree(got, want) || !reflect.DeepEqual(gotError, wantError) {
				t.Fatalf("AST differs: %t; diagnostics: %v, want %v", !sameGoTree(got, want),
					gotError, wantError)
			}
		})
	}
}

// TestPostfixGrammarChecksOperand checks that suffixes attach only to the current call.
func TestPostfixGrammarChecksOperand(t *testing.T) {
	for _, expression := range []string{"f()!!!", "f()! !", "f()!/* split */!", "f().(error)!"} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo",
				"package p; var value="+expression, SkipObjectResolution)
			if err == nil {
				t.Fatalf("invalid expression parsed: %s", expression)
			}
		})
	}
}

// TestPostfixGrammarRecordsRawCalls checks operand identity and raw AST before normalization.
func TestPostfixGrammarRecordsRawCalls(t *testing.T) {
	input := "package p\nfunc f() { _ = factory()!()!! }"
	set := token.NewFileSet()
	file, err := ParseTGoFile(set, "main.tgo", "main.tgo", input, SkipObjectResolution)
	forms := syntaxNodes(file)
	if err != nil || len(forms.Postfix) != 2 {
		t.Fatalf("operands: %v; error: %v", forms.Postfix, err)
	}
	for _, operation := range forms.Postfix {
		suffix := input[set.Position(operation.OpPos).Offset:set.Position(operation.OpEnd).Offset]
		if operation.Wrap != (suffix == "!!") || (suffix != "!" && suffix != "!!") {
			t.Fatalf("suffix: %q; operation: %+v", suffix, operation)
		}
	}
	outer := forms.Postfix[1].X.(*ast.CallExpr)
	if outer.Fun != forms.Postfix[0] {
		t.Fatal("parser changed the raw call operand")
	}
	assignment := file.Decls[0].(*ast.FuncDecl).Body.List[0].(*ast.AssignStmt)
	if assignment.Rhs[0] != forms.Postfix[1] {
		t.Fatal("recorded operand is not the parsed AST node")
	}
}

// TestOrdinaryModeRejectsTGo checks that Go parsing does not enable TGo grammar from a filename.
func TestOrdinaryModeRejectsTGo(t *testing.T) {
	for _, source := range []string{
		"func f() error { call()!;return nil }",
		"func f(ok bool) int {return if ok {1}else{2}}",
		"var values=[]int{x for x:=range 3}",
		"func f() (int,error) {return ..., err}",
	} {
		input := "package p; " + source
		if _, err := ParseFile(token.NewFileSet(), "main.tgo", input, SkipObjectResolution); err == nil {
			t.Fatalf("Go mode accepted TGo source: %s", source)
		}
		if _, err := ParseTGoFile(token.NewFileSet(), "main.go", "main.go", input, SkipObjectResolution); err != nil {
			t.Fatalf("TGo mode rejected source because of its filename: %v", err)
		}
	}
}
