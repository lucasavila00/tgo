package parser

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// TestValueGrammar checks direct grammar parsing of nested values and Go headers.
func TestValueGrammar(t *testing.T) {
	for _, expression := range []string{
		`if ready { audit(); 1 } else { 2 }`,
		`if value := read(); value > 0 { value } else if ready { 2 } else { 3 }`,
		`if if ready { true } else { false } { 1 } else { 2 }`,
		`if if ready { values := Numbers{1, 2}; len(values)>0 } else { false } { 1 } else { 2 }`,
		`switch if ready { values := Numbers{1, 2}; len(values) } else { 0 } { case 1: 2; default: 3 }`,
		`switch value := read(); value { case 1: audit(); 2; default: 3 }`,
		`switch { case ready: 1; default: 2 }`,
		`switch value := input.(type) { case int: value; default: 0 }`,
		`switch read() { case if ready { 1 } else { 2 }: 3; default: 4 }`,
		`if ready { switch input { case 1: 2; default: 3 } } else { 4 }`,
		`(if ready { 1 } else { 2 }) + (switch input { case 1: 3; default: 4 })`,
	} {
		t.Run(expression, func(t *testing.T) {
			input := "package p\nfunc f() { use(" + expression + ") }"
			positions := token.NewFileSet()
			file, forms, err := ParseTGoFile(positions, "main_tgo.go", "main.tgo", input,
				ParseComments|AllErrors|SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if file == nil || len(forms.Conditionals) == 0 {
				t.Fatal("missing value syntax")
			}
			for _, form := range forms.Conditionals {
				checkConditionalContainer(t, positions, input, form)
			}
		})
	}
}

// checkConditionalContainer checks the syntax body and exact original source span.
func checkConditionalContainer(t *testing.T, positions *token.FileSet, input string, form Conditional) {
	t.Helper()
	if name := form.Call.Fun.(*ast.Ident).Name; name != "" {
		t.Fatalf("parser assigned a helper name: %q", name)
	}
	function, ok := form.Call.Args[0].(*ast.FuncLit)
	if !ok || len(function.Body.List) != 1 {
		t.Fatal("missing conditional statement container")
	}
	start, end := positions.Position(form.Position), positions.Position(form.Call.End())
	if start.Filename != "main.tgo" || input[end.Offset-1] != '}' {
		t.Fatalf("source span: %v to %v", start, end)
	}
	if !strings.HasPrefix(input[start.Offset:], "if") && !strings.HasPrefix(input[start.Offset:], "switch") {
		t.Fatalf("conditional start: %v", start)
	}
}

// TestValueGrammarKeepsStatements checks that ordinary if and switch do not create records.
func TestValueGrammarKeepsStatements(t *testing.T) {
	input := `package p; func f() { if ready { use(1) } else { use(2) }; ` +
		`switch value { case 1: if ready { use(3) }; default: use(4) }; ` +
		`use(if ready { defer done(); return 1 } else { 2 }) }`
	_, forms, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input, SkipObjectResolution)
	if err != nil || len(forms.Conditionals) != 1 {
		t.Fatalf("conditional records: %d; error: %v", len(forms.Conditionals), err)
	}
}

// TestSyntaxContainerIncludesPostfix checks source spans before postfix AST normalization.
func TestSyntaxContainerIncludesPostfix(t *testing.T) {
	for _, body := range []string{
		`return ..., call()!!`,
		`return ..., first() + call()!`,
		`return []int{call()!! for range 1}`,
		`return map[int]int{1: first() + call()! for range 1}`,
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p; func f() { " + body + " }"
			_, forms, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input, SkipObjectResolution)
			if err != nil || len(forms.Postfix) != 1 {
				t.Fatalf("postfix records: %d; error: %v", len(forms.Postfix), err)
			}
			var call *ast.CallExpr
			if len(forms.ShortReturns) == 1 {
				call = forms.ShortReturns[0].Call
			} else {
				call = forms.Comprehensions[0].Output
			}
			if call.End() != forms.Postfix[0].End {
				t.Fatalf("container end %d; operator end %d", call.End(), forms.Postfix[0].End)
			}
		})
	}
}

// TestHeaderAssignmentKeepsNestedStatementBodies checks the lifetime of header assignment context.
func TestHeaderAssignmentKeepsNestedStatementBodies(t *testing.T) {
	input := "package p; func f() { if x := if ready { if flag { consume()\n" +
		"for range values { use(1) } }; 1 } else { 2 }; x>0 { use(x) } }"
	_, forms, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input,
		AllErrors|SkipObjectResolution)
	if err != nil || len(forms.Conditionals) != 1 || len(forms.Comprehensions) != 0 {
		t.Fatalf("value records: %d; collection records: %d; error: %v",
			len(forms.Conditionals), len(forms.Comprehensions), err)
	}
}
