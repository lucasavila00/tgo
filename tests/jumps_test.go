package tests

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestInsertJumps(t *testing.T) {
	cases := []struct{ name, code, target, want string }{
		{"labeled short declaration", `func run() { goto L; L: x:=right(); fmt.Print(x) }`, "right()", "RM1"},
		{"labeled var declaration", `func run() { goto L; L: var x=right(); fmt.Print(x) }`, "right()", "RM1"},
		{"backward declaration jump", `func run() { n:=0; L: x:=right(); fmt.Print(x); n++; if n<2 {goto L} }`,
			"right()", "RM1RM1"},

		{"nested declaration labels",
			`func run() { goto A; A: B: x:=right(); fmt.Print(x); if false {goto B} }`, "right()", "RM1"},

		{"skip generated locals", `func run() { var x int; goto Done; x=right(); Done: fmt.Print(x) }`,
			"right()", "0"},

		{"labeled continue", `func run() { L: for i:=0; i<2; i=next(i) { fmt.Print(i); continue L } }`,
			"next(i)", "0TM1TM"},

		{"goto and continue",
			`func run() { again:=false; L: for i:=0; i<2; i=next(i) { fmt.Print(i); if ` +
				`!again { again=true; goto L }; continue L } }`,
			"next(i)", "00TM1TM"},

		{"closure label scope",
			`func run() { again:=false; L: for i:=0;i<1;i++ {fn:=func(){L:for j:=0;j<1;` +
				`j=next(j){continue L}};fn();if !again {again=true;goto L};continue L} }`,
			"next(j)", "TMTM"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `package main
import "fmt"
func right() int { fmt.Print("R");return 1 }
func next(i int) int { fmt.Print("T");return i+1 }
func main() { run() }
` + tc.code
			actual := runInsertion(t, input, tc.target, func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"),
					Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING,
					Value: `"M"`}}}}}
			})
			if actual != tc.want {
				t.Fatalf("got %q, want %q", actual, tc.want)
			}
		})
	}
}
