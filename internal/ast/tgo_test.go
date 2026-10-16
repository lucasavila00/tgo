package ast

import (
	"go/token"
	"reflect"
	"testing"
)

// TestNativeNodeTraversal checks source-child identity and order through a collection's native nodes.
func TestNativeNodeTraversal(t *testing.T) {
	output := &PropagateExpr{X: &CallExpr{Fun: &Ident{Name: "output", NamePos: 10}, Lparen: 16,
		Rparen: 17}, OpPos: 18, OpEnd: 20, Wrap: true}
	rangeClause := &RangeClause{For: 21, Range: 29, Key: &Ident{Name: "key", NamePos: 25},
		Tok: token.DEFINE, X: &Ident{Name: "source", NamePos: 35}, EndPos: 41}
	filter := &FilterClause{If: 42, Cond: &Ident{Name: "filter", NamePos: 45}, EndPos: 51}
	collection := &ComprehensionExpr{Type: &Ident{Name: "Collection", NamePos: 1}, Lbrace: 9,
		Rbrace: 52, Output: []Expr{output}, Clauses: []ComprehensionClause{rangeClause, filter}}
	var visited []string
	seen := make(map[Node]bool)
	Inspect(collection, func(node Node) bool {
		seen[node] = true
		if identifier, ok := node.(*Ident); ok {
			visited = append(visited, identifier.Name)
		}
		return true
	})
	want := []string{"Collection", "output", "key", "source", "filter"}
	if !reflect.DeepEqual(visited, want) || !seen[output] || !seen[rangeClause] || !seen[filter] {
		t.Fatalf("visited names: %v; source-node identity: %v", visited, seen)
	}
	if collection.Pos() != 1 || collection.End() != 53 || output.End() != 20 {
		t.Fatal("source spans changed")
	}
}

// TestNativeControlFlowNodes checks branch children and return spans without a function container.
func TestNativeControlFlowNodes(t *testing.T) {
	shortened := &ShortReturnStmt{Return: 20, Placeholder: 27, PlaceholderEnd: 29,
		Results: []Expr{&Ident{Name: "err", NamePos: 30}}, ZeroSuccess: true}
	branch := &IfExpr{If: 1, Cond: &Ident{Name: "ready", NamePos: 4},
		Body: &BlockStmt{Lbrace: 10, List: []Stmt{shortened}, Rbrace: 35},
		Else: &BlockStmt{Lbrace: 42, Rbrace: 50}}
	statement := &TypeSwitchStmt{Switch: 60,
		Assign: &ExprStmt{X: &TypeAssertExpr{X: &Ident{Name: "value", NamePos: 67}, Lparen: 72, Rparen: 77}},
		Body:   &BlockStmt{Lbrace: 78, Rbrace: 80}}
	choice := &SwitchExpr{Switch: statement}
	seen := make(map[Node]bool)
	for _, expression := range []Expr{branch, choice} {
		Inspect(expression, func(node Node) bool {
			if _, function := node.(*FuncLit); function {
				t.Fatal("native control flow contains a function container")
			}
			seen[node] = true
			return true
		})
	}
	if !seen[shortened] || !seen[statement] || branch.End() != 51 || choice.End() != 81 || shortened.End() != 33 {
		t.Fatal("child identity or source end changed")
	}
	shortened.ZeroSuccess, shortened.Placeholder, shortened.PlaceholderEnd = false, 34, 36
	if shortened.End() != 36 {
		t.Fatal("trailing placeholder is outside the return span")
	}
}
