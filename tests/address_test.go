package tests

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"testing"
)

// TestInsertAddressPanic checks that taking an address can panic before the inserted return runs.
func TestInsertAddressPanic(t *testing.T) {
	for _, operation := range []string{"a[index()].M()", "_ = &a[index()]", "_ = &(a[index()])"} {
		t.Run(operation, func(t *testing.T) {
			input := `package main
import "fmt"
import "errors"
var failure=errors.New("failure")
type S struct{}
func(s *S) M(){}
func index()int{fmt.Print("I");return 1}
func run(err error)error{a:=[1]S{};` + operation + `;return nil}
func main(){defer func(){if recover()!=nil{fmt.Print("P")}}();fmt.Print(run(failure)==failure)}
`
			actual := runInsertion(t, input, "a[index()]", func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}}
			})
			if actual != "IP" {
				t.Fatalf("got %q, want IP", actual)
			}
		})
	}
}

// TestInsertArrayOperandPanicOrder checks array operand and index evaluation before an inserted return.
func TestInsertArrayOperandPanicOrder(t *testing.T) {
	for _, expression := range []string{"p.A[index()]", "a[bad()][index()]"} {
		t.Run(expression, func(t *testing.T) {
			input := `package main
import("fmt";"errors")
var failure=errors.New("failure")
type S struct{A [1]int}
func bad()int{fmt.Print("B");return 2}
func index()int{fmt.Print("I");return 0}
func run(err error)error{var p *S;var a [1][1]int;_=p;_=a;_=` + expression + `;return nil}
func main(){defer func(){if recover()!=nil{fmt.Print("P")}}();fmt.Print(run(failure)==failure)}
`
			actual := runInsertion(t, input, "index()", func([]ast.Expr) []ast.Stmt {
				return []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}}}
			})
			want := "Itrue"
			if expression == "a[bad()][index()]" {
				want = "BItrue"
			}
			if actual != want {
				t.Fatalf("got %q, want %q", actual, want)
			}
		})
	}
}
