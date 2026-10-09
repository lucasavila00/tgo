package syntax_test

import (
	"go/token"
	"testing"

	"tgo/pkg/syntax"
)

func TestParseFileComprehensionClausesKeepOrderAndPositions(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample
func collect(matrix [][]int) []int {
return []int{for row, values := range matrix {
for _, value := range values {
if value > row { value }
}
}}
}
`)
	files := token.NewFileSet()
	file, err := syntax.ParseFile(
		files, "clauses.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var comprehension *syntax.ComprehensionExpression = nil
	for _, extension := range syntax.Extensions(file) {
		if value, ok := syntax.ComprehensionExpressionOf(extension); ok {
			comprehension = value
		}
	}
	if comprehension == nil {
		t.Fatal("comprehension is absent")
	}
	if len(comprehension.Clauses) != 3 {
		t.Fatalf("clause count = %d, want 3", len(comprehension.Clauses))
	}

	outer, ok := syntax.ComprehensionRangeClauseOf(&comprehension.Clauses[0])
	if !ok {
		t.Fatal("first clause is not a range")
	}
	inner, ok := syntax.ComprehensionRangeClauseOf(&comprehension.Clauses[1])
	if !ok {
		t.Fatal("second clause is not a range")
	}
	filter, ok := syntax.ComprehensionFilterClauseOf(&comprehension.Clauses[2])
	if !ok {
		t.Fatal("third clause is not a filter")
	}
	requireComprehensionBindings(t, outer, []string{"row", "values"})
	requireComprehensionBindings(t, inner, []string{"_", "value"})

	positions := []struct {
		name string
		got  token.Pos
		line int
		col  int
	}{
		{name: "outer for", got: outer.For, line: 3, col: 14},
		{name: "outer start", got: outer.Start, line: 3, col: 14},
		{name: "outer stop", got: outer.Stop, line: 7, col: 2},
		{name: "outer define", got: outer.Define, line: 3, col: 30},
		{name: "outer range", got: outer.Range, line: 3, col: 33},
		{name: "outer open", got: outer.Lbrace, line: 3, col: 46},
		{name: "outer close", got: outer.Rbrace, line: 7, col: 1},
		{name: "inner for", got: inner.For, line: 4, col: 1},
		{name: "inner start", got: inner.Start, line: 4, col: 1},
		{name: "inner stop", got: inner.Stop, line: 6, col: 2},
		{name: "inner open", got: inner.Lbrace, line: 4, col: 30},
		{name: "inner close", got: inner.Rbrace, line: 6, col: 1},
		{name: "filter if", got: filter.If, line: 5, col: 1},
		{name: "filter start", got: filter.Start, line: 5, col: 1},
		{name: "filter stop", got: filter.Stop, line: 5, col: 25},
		{name: "filter open", got: filter.Lbrace, line: 5, col: 16},
		{name: "filter close", got: filter.Rbrace, line: 5, col: 24},
	}
	for _, item := range positions {
		position := files.Position(item.got)
		if position.Line != item.line || position.Column != item.col {
			t.Errorf(
				"%s position = %d:%d, want %d:%d",
				item.name, position.Line, position.Column, item.line, item.col,
			)
		}
	}
}

func TestParseFileRejectsComprehensionRangeAfterFilter(t *testing.T) {
	t.Parallel()
	source := []byte(`package sample
func collect(values []int) []int {
return []int{for _, value := range values {
if value > 0 {
for _, next := range values { next }
}
}}
}
`)
	_, err := syntax.ParseFile(
		token.NewFileSet(), "order.tgo", source, syntax.AllErrors,
	)
	want := "order.tgo:5:1: comprehension ranges must precede the filter"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}
