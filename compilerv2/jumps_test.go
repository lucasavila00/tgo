package compilerv2

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestInsertJumps(t *testing.T) {
	cases := []struct{ name, code, target, want string }{
		{"skip generated locals", `func run() { var x int; goto Done; x=right(); Done: fmt.Print(x) }`, "right()", "0"},
		{"labeled continue", `func run() { L: for i:=0; i<2; i=next(i) { fmt.Print(i); continue L } }`, "next(i)", "0TM1TM"},
		{"goto and continue", `func run() { again:=false; L: for i:=0; i<2; i=next(i) { fmt.Print(i); if !again { again=true; goto L }; continue L } }`, "next(i)", "00TM1TM"},
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
				return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
			})
			if actual != tc.want {
				t.Fatalf("got %q, want %q", actual, tc.want)
			}
		})
	}
}
