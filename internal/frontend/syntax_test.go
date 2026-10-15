package frontend

import "github.com/lucasavila00/tgo/internal/ast"

// conditionalExpressions returns value expressions and excludes structural else-if continuations.
func conditionalExpressions(file *ast.File) []ast.Expr {
	var expressions []ast.Expr
	continuations := make(map[*ast.IfExpr]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.IfExpr:
			if !continuations[node] {
				expressions = append(expressions, node)
			}
			if child, ok := node.Else.(*ast.IfExpr); ok {
				continuations[child] = true
			}
		case *ast.SwitchExpr:
			expressions = append(expressions, node)
		}
		return true
	})
	return expressions
}

// shortReturns returns the real shortened return statements in the source tree.
func shortReturns(file *ast.File) []*ast.ShortReturnStmt {
	var statements []*ast.ShortReturnStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if statement, ok := node.(*ast.ShortReturnStmt); ok {
			statements = append(statements, statement)
		}
		return true
	})
	return statements
}
