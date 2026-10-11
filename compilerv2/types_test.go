package compilerv2

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestInsertBooleanContext(t *testing.T) {
	for _, target := range []string{"left()==right()", "(left()==right()) && (right()==left())", "right()"} {
		t.Run(target, func(t *testing.T) {
			input := `package main
import "fmt"
type Flag bool
func left() int { fmt.Print("L");return 1 }
func right() int { fmt.Print("R");return 1 }
func use(v Flag) { fmt.Print("U") }
func main() { use((left()==right()) && (right()==left())) }
`
			actual := runInsertion(t, input, target, func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
			})
			want := "LRMRLU"
			if target == "(left()==right()) && (right()==left())" {
				want = "LRRLMU"
			}
			if target == "right()" {
				want = "LRRMLU"
			}
			if actual != want {
				t.Fatalf("got %q, want %q", actual, want)
			}
		})
	}
}
