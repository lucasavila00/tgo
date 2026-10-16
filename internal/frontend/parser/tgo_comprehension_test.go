package parser

import (
	"github.com/lucasavila00/tgo/internal/ast"
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
			file, err := ParseTGoFile(positions, "main_tgo.go", "main.tgo", input,
				ParseComments|AllErrors|SkipObjectResolution)
			forms := syntaxNodes(file)
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
			if !sameGoTree(got, want) || !reflect.DeepEqual(gotError, wantError) {
				t.Fatalf("AST differs: %t; diagnostics: %v, want %v", !sameGoTree(got, want),
					gotError, wantError)
			}
		})
	}
}

// checkComprehensionContainer checks the nested ranges, filters, and output identity.
func checkComprehensionContainer(t *testing.T, positions *token.FileSet, form *ast.ComprehensionExpr) {
	t.Helper()
	if len(form.Clauses) == 0 || positions.Position(form.Pos()).Filename != "main.tgo" {
		t.Fatal("incomplete comprehension")
	}
	if _, rangeFirst := form.Clauses[0].(*ast.RangeClause); !rangeFirst {
		t.Fatal("first clause is not a range")
	}
	for _, clause := range form.Clauses {
		if clause.Pos() < form.Lbrace || clause.End() > form.End() {
			t.Fatal("clause is outside source span")
		}
	}
}

// TestComprehensionGrammarRejectsMalformedClauses checks syntax errors before checking types.
func TestComprehensionGrammarRejectsMalformedClauses(t *testing.T) {
	for _, expression := range []string{
		`[]int{x for i := 0; i < 3; i++}`,
		`[]int{x for a, b, c := range xs}`,
		`[]int{x for _, x := range xs if}`,
		`[]int{x for _, x := range xs, y}`,
		`[]int{x for _, x := range xs`,
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo",
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
		`for range Table[int, int]{x, x*2 for x := range 3} { use(1) }`,
		`for range Numbers{x, for x := range 3} { use(1) }`,
		`for range Numbers{for range 3} { use(1) }`,
		`if xs := Table[int, int]{x, x*2, for x := range 3}; len(xs)>0 { use(1) }`,
		`if xs := Numbers{for range 3}; len(xs)>0 { use(1) }`,
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
			file, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input,
				AllErrors|SkipObjectResolution)
			forms := syntaxNodes(file)
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
			file, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo", input, mode)
			forms := syntaxNodes(file)
			if err != nil || len(forms.Comprehensions) != 0 {
				t.Fatalf("collections: %d; error: %v", len(forms.Comprehensions), err)
			}
			want, wantError := stock.ParseFile(token.NewFileSet(), "main.tgo", input, stock.Mode(mode))
			if !sameGoTree(file, want) || !reflect.DeepEqual(err, wantError) {
				t.Fatalf("ordinary statement tree or diagnostic changed: %v, want %v", err, wantError)
			}
		})
	}
}

// TestComprehensionOutputFacts keeps output count decisions in the checker.
func TestComprehensionOutputFacts(t *testing.T) {
	for _, test := range []struct {
		expression string
		count      int
	}{
		{"[]int{for range 3}", 0},
		{"[]int{x for x := range 3}", 1},
		{"[]int{x, for x := range 3}", 1},
		{"[]int{1,x for x := range 3}", 2},
		{"[]int{1,x,2 for x := range 3}", 3},
		{"map[int]int{x, x*2 for x := range 3}", 2},
		{"map[int]int{x: x*2 for x := range 3}", 2},
		{"map[int]int{x:x*2, 1:2 for x := range 3}", 4},
	} {
		t.Run(test.expression, func(t *testing.T) {
			file, err := ParseTGoFile(token.NewFileSet(), "main.tgo", "main.tgo",
				"package p; var values="+test.expression, AllErrors|SkipObjectResolution)
			forms := syntaxNodes(file)
			if err != nil || len(forms.Comprehensions) != 1 {
				t.Fatalf("source collections: %+v; error: %v", forms.Comprehensions, err)
			}
			if got := outputCount(forms.Comprehensions[0].Output); got != test.count {
				t.Fatalf("output count: %d; want %d", got, test.count)
			}
		})
	}
}

// TestClauseOperandShape checks real range and filter operands with complete postfix spans.
func TestClauseOperandShape(t *testing.T) {
	input := "package p; var result=[]int{x for x := range source()!! if keep(x)!}"
	positions := token.NewFileSet()
	file, err := ParseTGoFile(positions, "main.tgo", "main.tgo", input, AllErrors|SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	collection := syntaxNodes(file).Comprehensions[0]
	rangeClause := collection.Clauses[0].(*ast.RangeClause)
	filter := collection.Clauses[1].(*ast.FilterClause)
	for _, expression := range []ast.Expr{rangeClause.X, filter.Cond} {
		if _, ok := expression.(*ast.PropagateExpr); !ok {
			t.Fatalf("source operand: %T", expression)
		}
		end := positions.Position(expression.End()).Offset
		if input[end-1] != '!' {
			t.Fatalf("operand span excludes suffix: %v", expression)
		}
	}
}

// outputCount counts the written output values without changing key-value source nodes.
func outputCount(output []ast.Expr) int {
	count := 0
	for _, expression := range output {
		count++
		if _, keyValue := expression.(*ast.KeyValueExpr); keyValue {
			count++
		}
	}
	return count
}
