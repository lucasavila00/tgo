package frontend

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"
	"strings"
	"testing"
)

// TestParseShortReturns checks shortened returns with tuples, comments, propagation, and nested expressions.
func TestParseShortReturns(t *testing.T) {
	for _, body := range []string{
		"return ..., err",
		"return value, ...",
		"return x, y, ...",
		"return pair(), ...",
		"return ...,\n err",
		"return ..., // before\n err // after\n",
		"return ..., err /* after */",
		"return ..., err // after\n",
		"return value, ... /* after\n */ println(1)",
		"return value, ...\n println(1)",
		"return call()!, ...",
		"return if ready { return ..., err; 1 } else { 2 }, ...",
		"return []int{if ready {return ..., err; 1} else {2} for range 1}, ...",
		"return `raw\r\n..`, ...",
		"return value /* .. */, ... // comment\n println(1)",
	} {
		t.Run(body, func(t *testing.T) {
			checkShortReturnPositions(t, body)
		})
	}
}

// TestRejectShortReturnSyntax checks that invalid ellipsis forms return diagnostics with input positions.
func TestRejectShortReturnSyntax(t *testing.T) {
	for _, body := range []string{
		"return ...,", "return ..., err, ...", "return x, ... y", "return x, ..., y",
		"return ... err", "return value ...", "return ..., ..., err", "return .. err", "return value ..",
	} {
		t.Run(body, func(t *testing.T) {
			_, err := ParseFile(
				token.NewFileSet(),
				"main_tgo.go",
				"main.tgo",
				[]byte("package p\nfunc f() error { "+body+" }"),
			)
			if err == nil || !strings.Contains(err.Error(), "main.tgo:") {
				t.Fatalf("source diagnostic: %v", err)
			}
		})
	}
}

// TestShortReturnLeavesOrdinaryGo checks that dots in strings, comments, numbers, and variadic calls stay unchanged.
func TestShortReturnLeavesOrdinaryGo(t *testing.T) {
	input := "package p\nfunc f(xs ...float64) float64 { _ = `..`; /* .. */ return g(xs...) + 1.5 }"
	parsed, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input))
	if err != nil || len(shortReturns(parsed.File)) != 0 {
		t.Fatalf("ordinary syntax changed: %v", err)
	}
}

// checkShortReturnPositions parses a return body and requires one source-position check per source return.
func checkShortReturnPositions(t *testing.T, body string) {
	t.Helper()
	input := "package p\nfunc f() (int, error) { " + body + "\n}"
	set := token.NewFileSet()
	parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(shortReturns(parsed.File)) == 0 {
		t.Fatal("missing shortened return")
	}
	seen := 0
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		seen += checkShortReturnPosition(t, input, set, node)
		return true
	})
	if seen != len(shortReturns(parsed.File)) {
		t.Fatalf("%d checked returns, %d source returns", seen, len(shortReturns(parsed.File)))
	}
}

// checkShortReturnPosition checks the ... source span and operand bounds of one shortened return.
// It returns one for a checked return and zero for other nodes.
func checkShortReturnPosition(t *testing.T, input string, set *token.FileSet, node ast.Node) int {
	t.Helper()
	statement, ok := node.(*ast.ShortReturnStmt)
	if !ok {
		return 0
	}
	position := set.Position(statement.Placeholder)

	if input[position.Offset:position.Offset+3] != "..." || position.Filename != "main.tgo" {
		t.Fatalf("operator position: %v", position)
	}
	for _, operand := range statement.Results {
		if operand.Pos() < statement.Pos() || operand.End() > statement.End() {
			t.Fatal("operand outside return span")
		}
	}
	return 1
}
