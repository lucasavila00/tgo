package tests

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestInsertOmittedLiteralType(t *testing.T) {
	for _, target := range []string{"{value()}", "value()"} {
		t.Run(target, func(t *testing.T) {
			input := `package main
import "fmt"
type Key struct{N int}
func value()int{fmt.Print("V");return 1}
func main(){m:=map[Key]int{{value()}:value()};fmt.Print(m[Key{1}])}
`
			actual := runInsertion(t, input, target, func(results []ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
			})
			want := "VMV1"
			if target == "value()" {
				want = "VVM1"
			}
			if actual != want {
				t.Fatalf("got %q, want %q", actual, want)
			}
		})
	}
}

func TestInsertOmittedLiteralContexts(t *testing.T) {
	for _, test := range []struct{ name, code, target, want string }{
		{"slice", `a:=[][]int{{value()}};fmt.Print(a)`, "{value()}", "VM[[1]]"},
		{"map value", `a:=map[int][]int{0:{value()}};fmt.Print(a[0])`, "{value()}", "VM[1]"},
		{"pointer key", `a:=map[*Key]int{{value()}:2};for k:=range a{fmt.Print(k.N)}`, "{value()}", "VM1"},
		{"pointer map", `a:=map[int]*map[Key]int{0:{{value()}:2}};fmt.Print(len(*a[0]))`, "{value()}", "VM1"},
		{"pointer key in pointer map", `a:=map[int]*map[*Key]int{0:{{value()}:2}};for k:=range *a[0]{fmt.Print(k.N)}`, "{value()}", "VM1"},
		{"array key", `a:=map[[20]int]int{{12:value(),value()}:2};fmt.Print(len(a))`, "{12:value(),value()}", "VVM1"},
		{"nested", `a:=[][][]int{{{value()}}};fmt.Print(a)`, "{value()}", "VM[[[1]]]"},
		{"key panic", `defer func(){if recover()!=nil{fmt.Print("P")}}();_=map[Bad]int{{bad()}:value()}`, "{bad()}", "BMVP"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := `package main
import "fmt"
type Key struct{N int}
type Bad struct{Value any}
func value()int{fmt.Print("V");return 1}
func bad()any{fmt.Print("B");return []int{1}}
func main(){` + test.code + `}`
			actual := runInsertion(t, input, test.target, func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
			})
			if actual != test.want {
				t.Fatalf("got %q, want %q", actual, test.want)
			}
		})
	}
}

func TestInsertPrivateLiteralType(t *testing.T) {
	input := `package main
import("fmt";"example.com/insertion/foreign")
func value()int{fmt.Print("V");return 1}
func main(){a:=foreign.Map{{value()}:value()};fmt.Print(len(a))}
`
	actual := runInsertionFiles(t, input, "{value()}", func(results []ast.Expr) []ast.Stmt {
		return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}, &ast.SelectorExpr{X: results[0], Sel: ast.NewIdent("N")}}}}}
	}, map[string]string{"foreign/foreign.go": "package foreign\ntype key struct{N int}\ntype Map map[key]int\n"})
	if actual != "VM1V1" {
		t.Fatalf("got %q", actual)
	}
}

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

func TestInsertEarlierContext(t *testing.T) {
	input := `package main
import "fmt"
type Flag bool
func value() int { fmt.Print("V");return 1 }
func use(v Flag,p *int,n int) { fmt.Print("U") }
func main() { use(value()==1,nil,value()) }
`
	actual := runInsertion(t, input, "value()", func([]ast.Expr) []ast.Stmt {
		return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("fmt"), Sel: ast.NewIdent("Print")}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"M"`}}}}}
	})
	if actual != "VVMU" {
		t.Fatalf("got %q, want VVMU", actual)
	}
}
