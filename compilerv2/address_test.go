package compilerv2

import (
	"go/ast"
	"testing"
)

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
