package compilerv2

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestInsertStorage(t *testing.T) {
	cases := []struct{ name, code, target, want string }{
		{"array slice", `func main() { a:=[2]int{1,2}; s:=a[:bound()]; s[0]=9; fmt.Print(a[0]) }`, "bound()", "BM9"},
		{"array index", `var a=[2]int{1,2}; func index() int { a[0]=9;fmt.Print("I");return 0 };func main() { fmt.Print(a[index()]) }`, "index()", "IM9"},
		{"pointer receiver", `type S struct { n int };func(s *S) Add(v int) { s.n+=v };func main() { a:=[1]S{};a[bound()-1].Add(bound());fmt.Print(a[0].n) }`, "bound()", "BBM1"},
		{"receiver index", `type S struct { n int };func(s *S) Add(v int) { s.n+=v };func index() int { fmt.Print("I");return 0 };func main() { a:=[1]S{};a[index()].Add(bound());fmt.Print(a[0].n) }`, "index()", "IMB1"},
		{"address literal", `type S struct{n int};func main(){p:=&S{n:bound()};fmt.Print(p.n)}`, "bound()", "BM1"},
		{"address literal root", `type S struct{n int};func main(){p:=&S{n:bound()};fmt.Print(p.n)}`, "S{n:bound()}", "BM1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := `package main
import "fmt"
func bound() int { fmt.Print("B");return 1 }
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
