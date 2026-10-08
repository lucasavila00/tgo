package compiler

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestParseSourceLowersNestedMatchLabels(t *testing.T) {
	data := []byte(`package sample
type E enum { A struct{} }
func use(value E) {
Outer: Inner: match value { case A(item): break Outer }
}
`)
	source, err := parseSource(token.NewFileSet(), "sample.tgo", data)
	if err != nil {
		t.Fatal(err)
	}
	labels := 0
	var outer *ast.LabeledStmt
	ast.Inspect(source.File, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.LabeledStmt:
			labels++
			if node.Label.Name == "Outer" {
				outer = node
			}
		case *ast.EmptyStmt:
			if !node.Implicit {
				t.Fatalf("artificial statement at %v", node.Pos())
			}
		}
		return true
	})
	if outer == nil {
		t.Fatalf("lowered labels=%d outer is missing", labels)
	}
	inner, ok := outer.Stmt.(*ast.LabeledStmt)
	if !ok || inner.Label.Name != "Inner" {
		t.Fatalf("lowered labels=%d outer=%#v", labels, outer)
	}
	if _, ok := inner.Stmt.(*ast.SwitchStmt); !ok {
		t.Fatalf("inner label statement: %T", inner.Stmt)
	}
}
