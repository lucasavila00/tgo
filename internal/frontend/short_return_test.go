package frontend

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestParseShortReturns(t *testing.T) {
	for _, body := range []string{
		"return .. err",
		"return value ..",
		"return x, y ..",
		"return pair() ..",
		"return ..\n err",
		"return .. // before\n err // after\n",
		"return .. err /* after */",
		"return .. err // after\n",
		"return value .. /* after\n */ println(1)",
		"return value ..\n println(1)",
		"return call()! ..",
		"return if ready { return .. err; 1 } else { 2 } ..",
		"return []int{if ready {return .. err; 1} else {2} for range 1} ..",
		"return `raw\r\n..` ..",
		"return value /* .. */ .. // comment\n println(1)",
	} {
		t.Run(body, func(t *testing.T) {
			checkShortReturnPositions(t, body)
		})
	}
}

func TestRejectShortReturnSyntax(t *testing.T) {
	for _, body := range []string{"return ..", "return .. err ..", "return x .. y", "return x .., y", "return .., err"} {
		t.Run(body, func(t *testing.T) {
			_, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte("package p\nfunc f() error { "+body+" }"), make(map[string]bool))
			if err == nil || !strings.Contains(err.Error(), "main.tgo:") {
				t.Fatalf("source diagnostic: %v", err)
			}
		})
	}
}

func TestShortReturnLeavesOrdinaryGo(t *testing.T) {
	input := "package p\nfunc f(xs ...float64) float64 { _ = `..`; /* .. */ return g(xs...) + 1.5 }"
	parsed, err := ParseFile(token.NewFileSet(), "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil || len(parsed.ShortReturns) != 0 {
		t.Fatalf("ordinary syntax changed: %v", err)
	}
}

func checkShortReturnPositions(t *testing.T, body string) {
	t.Helper()
	input := "package p\nfunc f() (int, error) { " + body + "\n}"
	set := token.NewFileSet()
	parsed, err := ParseFile(set, "main_tgo.go", "main.tgo", []byte(input), make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ShortReturns) == 0 {
		t.Fatal("missing return record")
	}
	seen := 0
	ast.Inspect(parsed.File, func(node ast.Node) bool {
		seen += checkShortReturnPosition(t, input, parsed, node)
		return true
	})
	if seen != len(parsed.ShortReturns) {
		t.Fatalf("%d returns, %d records", seen, len(parsed.ShortReturns))
	}
}

func checkShortReturnPosition(t *testing.T, input string, parsed *Parsed, node ast.Node) int {
	t.Helper()
	statement, ok := node.(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return 0
	}
	call, ok := statement.Results[0].(*ast.CallExpr)
	if !ok {
		return 0
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return 0
	}
	record, ok := parsed.ShortReturns[id.Name]
	if !ok {
		return 0
	}

	if input[record.Position.Offset:record.Position.Offset+2] != ".." || record.Position.Filename != "main.tgo" {
		t.Fatalf("operator position: %v", record.Position)
	}
	for _, operand := range call.Args {
		if operand.Pos() < statement.Pos() || operand.End() > statement.End() {
			t.Fatal("operand outside return span")
		}
	}
	return 1
}
