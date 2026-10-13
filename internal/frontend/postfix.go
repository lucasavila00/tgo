package frontend

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"

	"github.com/lucasavila00/tgo/internal/frontend/parser"

	"golang.org/x/tools/go/ast/astutil"
)

// Marker records a postfix operation and its original source position.
type Marker struct {
	Name     string
	Wrap     bool
	CallName string
	Position token.Position
}

type postfixBinding struct {
	name string
	end  token.Pos
}

// bindPostfixCalls records source names before any nested operand is replaced.
func (result *Parsed) bindPostfixCalls(fset *token.FileSet, forms parser.Forms,
	input []byte, used map[string]bool) map[ast.Expr]postfixBinding {
	bindings := make(map[ast.Expr]postfixBinding, len(forms.Postfix))
	containers := syntaxContainers(forms)
	for _, operation := range forms.Postfix {
		call := ast.Unparen(operation.Operand).(*ast.CallExpr)
		name := freshMarkerName(used, "tgoPropagation")
		result.Propagations[name] = Marker{
			Name: name, Wrap: operation.Wrap, Position: fset.Position(operation.Start),
			CallName: sourceCallName(fset, call.Fun, input, containers),
		}
		bindings[operation.Operand] = postfixBinding{name: name, end: operation.End}
	}
	return bindings
}

// sourceCallName formats ordinary callees and retains source syntax for added forms.
func sourceCallName(fset *token.FileSet, expression ast.Expr, input []byte, containers map[ast.Node]bool) string {
	containsSyntax := false
	ast.Inspect(expression, func(node ast.Node) bool {
		containsSyntax = containsSyntax || containers[node]
		return !containsSyntax
	})
	if containsSyntax {
		file := fset.File(expression.Pos())
		return string(input[file.Offset(expression.Pos()):file.Offset(expression.End())])
	}
	var name bytes.Buffer
	if err := format.Node(&name, fset, expression); err != nil {
		panic(err)
	}
	return name.String()
}

// normalizePostfix replaces recorded operands with checker calls in one bottom-up walk.
// Nested replacements are complete before a parent operand is wrapped.
func normalizePostfix(file *ast.File, bindings map[ast.Expr]postfixBinding) {
	astutil.Apply(file, nil, func(cursor *astutil.Cursor) bool {
		operand, isExpression := cursor.Node().(ast.Expr)
		if !isExpression {
			return true
		}
		binding, found := bindings[operand]
		if found {
			cursor.Replace(&ast.CallExpr{
				Fun:    &ast.Ident{Name: binding.name, NamePos: operand.Pos()},
				Lparen: operand.Pos(), Args: []ast.Expr{ast.Unparen(operand)}, Rparen: binding.end - 1,
			})
		}
		return true
	})
}
