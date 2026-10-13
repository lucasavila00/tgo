package parser

import (
	"go/ast"
	stock "go/parser"
	"go/token"
	"reflect"
	"testing"
)

// TestComprehensionGrammar checks output-first syntax and ordered nested clauses.
func TestComprehensionGrammar(t *testing.T) {
	for _, expression := range []string{
		`[]int{x for _, x := range xs}`,
		`map[int]string{x: f(x) for _, x := range xs if x > 0}`,
		`Named[int]{x+y for _, x := range xs if x>0 for _, y := range ys if y>0}`,
		`[]int{1 for range 3}`,
		`[]int{x for x := range 3}`,
		`[]int{x for _, x := range []int{y for _, y := range ys}}`,
		`[]int{if x > 0 { x } else { 0 } for _, x := range if ready { xs } else { ys }}`,
		"[]int{\n x // output\n for _, x := range xs // source\n if x>0,\n}",
		`map[int]int{switch x {case 1: 2; default: 3}: next(x)! for x := range 3}`,
	} {
		t.Run(expression, func(t *testing.T) {
			input := "package p\nvar result = " + expression
			positions := token.NewFileSet()
			file, forms, err := ParseTGoFile(positions, "main_tgo.go", "main.tgo", input,
				ParseComments|AllErrors|SkipObjectResolution)
			if err != nil || file == nil || len(forms.Comprehensions) == 0 {
				t.Fatalf("collections: %d; error: %v", len(forms.Comprehensions), err)
			}
			for _, form := range forms.Comprehensions {
				checkComprehensionContainer(t, positions, form)
			}
		})
	}
}

// TestLiteralGrammarMatchesGo checks unchanged literal trees and invalid comma diagnostics.
func TestLiteralGrammarMatchesGo(t *testing.T) {
	for _, expression := range []string{
		`[]int{}`,
		`[]int{1, 2, 3}`,
		`map[string][]int{"first": {1, 2}, "second": {3}}`,
		`[]struct{ X int }{{X: 1}, {X: 2}}`,
		"[]int{\n1, // first\n2,\n}",
		"[]string{`a\r\nb`, /* kept */ \"c\"}",
		"[]int{1\n2}",
		"[]int{1\n}",
		"[]int{1;2}",
	} {
		t.Run(expression, func(t *testing.T) {
			input := "package p\nvar result = " + expression
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

// checkComprehensionContainer checks the nested ranges, filters, and output identity.
func checkComprehensionContainer(t *testing.T, positions *token.FileSet, form Comprehension) {
	t.Helper()
	if len(form.Call.Args) != 2 || len(form.Ranges) == 0 || positions.Position(form.Position).Filename != "main.tgo" {
		t.Fatal("incomplete comprehension record")
	}
	body := form.Call.Args[1].(*ast.FuncLit).Body
	rangeIndex := 0
	for len(body.List) == 1 {
		switch statement := body.List[0].(type) {
		case *ast.RangeStmt:
			if form.Ranges[rangeIndex] != statement {
				t.Fatal("range order differs from syntax order")
			}
			rangeIndex++
			body = statement.Body
		case *ast.IfStmt:
			body = statement.Body
		case *ast.ExprStmt:
			if statement.X != form.Output || rangeIndex != len(form.Ranges) {
				t.Fatal("output or range identity changed")
			}
			return
		default:
			t.Fatalf("unexpected clause: %T", statement)
		}
	}
	t.Fatal("missing output")
}

// TestComprehensionGrammarRejectsMalformedClauses checks syntax errors before checking types.
func TestComprehensionGrammarRejectsMalformedClauses(t *testing.T) {
	for _, expression := range []string{
		`[]int{x, y for _, x := range xs}`,
		`[]int{x for i := 0; i < 3; i++}`,
		`[]int{x for a, b, c := range xs}`,
		`[]int{x for _, x := range xs if}`,
		`[]int{x for _, x := range xs, y}`,
		`[]int{x for _, x := range xs`,
	} {
		t.Run(expression, func(t *testing.T) {
			_, _, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo",
				"package p\nvar result = "+expression, AllErrors|SkipObjectResolution)
			if err == nil {
				t.Fatal("invalid comprehension parsed")
			}
		})
	}
}

// TestNamedComprehensionHeaders checks collections beside ordinary statement braces.
func TestNamedComprehensionHeaders(t *testing.T) {
	for _, body := range []string{
		`for range Numbers{x for x := range 3} { use(1) }`,
		`for range Numbers[int]{x for x := range 3} { use(1) }`,
		`for range Table[int, string]{x: "value" for x := range 3} { use(1) }`,
		`if Flags{true for range 1}[0] { use(1) }`,
		`if xs := Numbers{x for x := range 3}; len(xs)>0 { use(1) }`,
		`if xs := Numbers{x for x := range Other{1, 2}}; len(xs)>0 { use(1) }`,
		`if xs := Numbers{if x := if ready { 1 } else { 2 }; x>0 { x } else { 0 } ` +
			`for range Other{1, 2}}; len(xs)>0 { use(1) }`,
		`switch len(Numbers{x for x := range 3}) { case 3: use(1) }`,
		`for value := range Numbers{x for x := range 3} { use(value) }`,
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p; func f() { " + body + " }"
			_, forms, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input,
				AllErrors|SkipObjectResolution)
			if err != nil || len(forms.Comprehensions) != 1 {
				t.Fatalf("collections: %d; error: %v", len(forms.Comprehensions), err)
			}
		})
	}
}

// TestNamedHeaderKeepsOrdinaryBlocks checks loops inside statement bodies.
func TestNamedHeaderKeepsOrdinaryBlocks(t *testing.T) {
	for _, body := range []string{
		`if ready { consume(); for range 3 { use(1) } }`,
		"if ready { consume()\nfor range 3 { use(1) } }",
		"if ready { consume()\nfor range values { use(1) } }",
		"if ready { defer consume()\nfor range values { use(1) } }",
		"if ready { func() { consume() }()\nfor range values { use(1) } }",
		"if ready { return\nfor range values { use(1) } }",
		`for range values { consume(); for range 3 { use(1) } }`,
		"for range values { consume()\nfor range other { use(1) } }",
		`if xs := func() []int { if ready { consume(); for range values { use(1) } }; ` +
			`return nil }(); len(xs)>0 { use(1) }`,
		`if ready { for range 3 { use(1) } }; next()`,
	} {
		t.Run(body, func(t *testing.T) {
			input := "package p; func f() { " + body + " }"
			mode := AllErrors | SkipObjectResolution
			file, forms, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input, mode)
			if err != nil || len(forms.Comprehensions) != 0 {
				t.Fatalf("collections: %d; error: %v", len(forms.Comprehensions), err)
			}
			want, wantError := stock.ParseFile(token.NewFileSet(), "main.tgo", input, stock.Mode(mode))
			if !reflect.DeepEqual(file, want) || !reflect.DeepEqual(err, wantError) {
				t.Fatalf("ordinary statement tree or diagnostic changed: %v, want %v", err, wantError)
			}
		})
	}
}
