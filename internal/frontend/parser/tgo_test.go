package parser

import (
	"github.com/lucasavila00/tgo/internal/ast"
	stockast "go/ast"
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

// TestTGoParserLeavesBindingToChecker checks that native parsing always returns syntax without legacy objects.
func TestTGoParserLeavesBindingToChecker(t *testing.T) {
	input := "package p; func f(x int, xs []int) []int { _ = if x := 1; x > 0 { x } else { x }; " +
		"_ = x; return []int{x for _, x := range xs if x > 0} }"
	for _, mode := range []Mode{0, AllErrors, SkipObjectResolution} {
		file, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input, mode)
		if err != nil {
			t.Fatal(err)
		}
		if file.Scope != nil || file.Unresolved != nil {
			t.Fatalf("mode %v: scope %v; unresolved %v", mode, file.Scope, file.Unresolved)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok && identifier.Obj != nil {
				t.Fatalf("mode %v: identifier %s has an object", mode, identifier.Name)
			}
			return true
		})
	}
}

// TestGoParserObjectModes checks that ordinary Go keeps its optional legacy resolution.
func TestGoParserObjectModes(t *testing.T) {
	input := "package p; func f(x int) int { return x }; var n = f(1)"
	for _, mode := range []Mode{0, AllErrors, SkipObjectResolution} {
		file, err := ParseFile(token.NewFileSet(), "main.go", input, mode)
		stockFile, stockError := stock.ParseFile(token.NewFileSet(), "main.go", input, stock.Mode(mode))
		fileTree, stockTree := *file, *stockFile
		fileTree.Scope, stockTree.Scope = nil, nil
		if !sameGoTree(&fileTree, &stockTree) || !reflect.DeepEqual(err, stockError) {
			t.Fatalf("mode %v: AST or diagnostics differ from Go", mode)
		}
		checkGoFileScope(t, file, stockFile)
	}
}

// checkGoFileScope compares each legacy package object without depending on map print order.
func checkGoFileScope(t *testing.T, file *ast.File, stockFile *stockast.File) {
	t.Helper()
	scope, stockScope := file.Scope, stockFile.Scope
	if scope == nil || stockScope == nil {
		if (scope == nil) != (stockScope == nil) {
			t.Fatal("package scope presence differs from Go")
		}
		return
	}
	if scope.Outer != nil || stockScope.Outer != nil || len(scope.Objects) != len(stockScope.Objects) {
		t.Fatal("package scope structure differs from Go")
	}
	for name, object := range scope.Objects {
		if !sameGoTree(object, stockScope.Objects[name]) {
			t.Fatalf("package object %s differs from Go", name)
		}
	}
}
