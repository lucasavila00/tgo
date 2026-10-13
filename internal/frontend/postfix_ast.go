package frontend

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// bindPostfixCalls finds the operand for each operator from its temporary comment.
// It records the source call name and panics if an operator does not follow a call.
func bindPostfixCalls(fset *token.FileSet, file *ast.File, operations map[string]Marker) map[ast.Expr]Marker {
	result := make(map[ast.Expr]Marker)
	for _, group := range file.Comments {
		for _, comment := range group.List {
			name := strings.TrimSuffix(strings.TrimPrefix(comment.Text, "/*"), "*/")
			operation, ok := operations[name]
			if !ok {
				continue
			}
			selected := precedingCall(file, comment.Pos())
			raw := ast.Unparen(selected)
			call, ok := raw.(*ast.CallExpr)
			if !ok {
				panic(fmt.Sprintf("%s: propagation requires a call", fset.Position(comment.Pos())))
			}
			var callName bytes.Buffer
			if err := format.Node(&callName, fset, call.Fun); err != nil {
				panic(err)
			}
			if operation.CallName == "" {
				operation.CallName = callName.String()
			}
			result[selected] = operation
		}
	}
	return result
}

// replacePostfixOperand replaces the selected expression with a marker call for source checking.
// It changes file in place and removes the comment for this operation.
func replacePostfixOperand(file *ast.File, original *token.File, expression ast.Expr, operation Marker) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		if cursor.Node() == expression {
			cursor.Replace(&ast.CallExpr{
				Fun:    &ast.Ident{Name: operation.Name, NamePos: expression.Pos()},
				Lparen: expression.Pos(), Args: []ast.Expr{ast.Unparen(expression)},
				Rparen: original.Pos(operation.EndOffset - 1),
			})
			return false
		}
		return true
	}, nil)
	comments := file.Comments[:0]
	for _, group := range file.Comments {
		kept := group.List[:0]
		for _, comment := range group.List {
			if comment.Text != "/*"+operation.Name+"*/" {
				kept = append(kept, comment)
			}
		}
		group.List = kept
		if len(kept) != 0 {
			comments = append(comments, group)
		}
	}
	file.Comments = comments
}

// precedingCall finds the call or parenthesized operand ending nearest to a postfix marker.
// For equal ends, it selects the expression with the earliest start.
func precedingCall(file *ast.File, position token.Pos) ast.Expr {
	var selected ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		switch expr.(type) {
		case *ast.CallExpr, *ast.ParenExpr:
		default:
			return true
		}
		if !ok || expr.End() > position {
			return true
		}
		if selected == nil || expr.End() > selected.End() || (expr.End() == selected.End() && expr.Pos() < selected.Pos()) {
			selected = expr
		}
		return true
	})
	return selected
}
