package parser

import (
	"bytes"
	"reflect"
	"sort"

	"github.com/lucasavila00/tgo/internal/ast"
)

// testSyntax collects source nodes to check identity, positions, and grammar counts.
type testSyntax struct {
	Postfix        []*ast.PropagateExpr
	Conditionals   []ast.Expr
	Comprehensions []*ast.ComprehensionExpr
	ShortReturns   []*ast.ShortReturnStmt
}

// syntaxNodes collects native forms and excludes an else-if continuation from the expression count.
func syntaxNodes(file ast.Node) testSyntax {
	var result testSyntax
	continuations := make(map[*ast.IfExpr]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.PropagateExpr:
			result.Postfix = append(result.Postfix, node)
		case *ast.IfExpr:
			if !continuations[node] {
				result.Conditionals = append(result.Conditionals, node)
			}
			if child, ok := node.Else.(*ast.IfExpr); ok {
				continuations[child] = true
			}
		case *ast.SwitchExpr:
			result.Conditionals = append(result.Conditionals, node)
		case *ast.ComprehensionExpr:
			result.Comprehensions = append(result.Comprehensions, node)
		case *ast.ShortReturnStmt:
			result.ShortReturns = append(result.ShortReturns, node)
		}
		return true
	})
	sort.Slice(result.Postfix, func(first, second int) bool {
		return result.Postfix[first].OpPos < result.Postfix[second].OpPos
	})
	return result
}

// sameGoTree compares Go syntax and requires ordinary case clauses despite private AST metadata.
func sameGoTree(first, second any) bool {
	var firstTree, secondTree bytes.Buffer
	ordinaryCases := true
	fields := func(name string, value reflect.Value) bool {
		if name == "Exhaustive" {
			ordinaryCases = ordinaryCases && !value.Bool()
			return false
		}
		return name != "TGo"
	}
	if ast.Fprint(&firstTree, nil, first, fields) != nil || ast.Fprint(&secondTree, nil, second, fields) != nil {
		return false
	}
	return ordinaryCases && bytes.Equal(firstTree.Bytes(), secondTree.Bytes())
}
