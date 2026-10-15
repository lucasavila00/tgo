package frontend

import (
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	"testing"
)

// TestParseComprehensions checks nested collections, filters, comments, and source nodes.
func TestParseComprehensions(t *testing.T) {
	for _, expression := range []string{
		`[]int{x for _, x := range xs}`,
		"[]string{`a\r\nb` for range 1}",
		"[]int{x /* output\r\ncomment */ for _,x := range xs /* source\r\ncomment */}",
		`map[int]string{x: f(x) for _, x := range xs if x > 0}`,
		`Named[int]{x+y for _, x := range xs for _, y := range ys}`,
		`[]int{x for _, x := range []int{y for _, y := range ys}}`,
		`[]int{if x > 0 {x} else {0} for _, x := range xs}`,
		`[]int{x for _, x := range if true {xs} else {ys} if x > 0}`,
		"[]int{\n x\n for _, x := range xs\n if x > 0,\n}",
		"[]int{x // output\n for _, x := range xs // source\n if x > 0 // filter\n}",
	} {
		t.Run(expression, func(t *testing.T) {
			set := token.NewFileSet()
			parsed, err := ParseFile(set, "main_tgo.go", "main.tgo",
				[]byte("package p\nvar result = "+expression))
			if err != nil {
				t.Fatal(err)
			}
			if len(comprehensions(parsed.File)) == 0 {
				t.Fatal("missing comprehension")
			}
			for _, record := range comprehensions(parsed.File) {
				if record.Type == nil || len(record.Clauses) == 0 || set.Position(record.Pos()).Filename != "main.tgo" {
					t.Fatalf("incomplete source node: %+v", record)
				}
			}
		})
	}
}

// TestComprehensionsKeepOrdinaryLoops checks that ordinary loops are not recorded as comprehensions.
func TestComprehensionsKeepOrdinaryLoops(t *testing.T) {
	input := []byte(`package p; func g(xs []int) []int { r := []int{}; for _,x := range xs { ` +
		`r=append(r,x) }; return r }; func f() { for range 2 { print(1); for range ` +
		`3 {} }; _ = []int{x for x := range 3} }`)
	parsed, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(comprehensions(parsed.File)) != 1 {
		t.Fatalf("got %d comprehensions", len(comprehensions(parsed.File)))
	}
}

// comprehensions returns the source collection expressions in traversal order.
func comprehensions(file *ast.File) []*ast.ComprehensionExpr {
	var expressions []*ast.ComprehensionExpr
	ast.Inspect(file, func(node ast.Node) bool {
		if expression, ok := node.(*ast.ComprehensionExpr); ok {
			expressions = append(expressions, expression)
		}
		return true
	})
	return expressions
}
