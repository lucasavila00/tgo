package frontend

import (
	"bytes"
	"go/token"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/printer"
)

// sourceCallNames records call text on the same propagation nodes used by checking and lowering.
func sourceCallNames(fset *token.FileSet, file *ast.File, input []byte) map[*ast.PropagateExpr]string {
	names := make(map[*ast.PropagateExpr]string)
	ast.Inspect(file, func(node ast.Node) bool {
		if operation, ok := node.(*ast.PropagateExpr); ok {
			operand := ast.Unparen(operation.X)
			for {
				nested, ok := operand.(*ast.PropagateExpr)
				if !ok {
					break
				}
				operand = ast.Unparen(nested.X)
			}
			names[operation] = sourceCallName(fset, operand.(*ast.CallExpr).Fun, input)
		}
		return true
	})
	return names
}

// sourceCallName prints a Go callee or retains its source text when it contains a value-producing form.
func sourceCallName(fset *token.FileSet, expression ast.Expr, input []byte) string {
	containsValueForm := false
	ast.Inspect(expression, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
			containsValueForm = true
		}
		return !containsValueForm
	})
	if containsValueForm {
		file := fset.File(expression.Pos())
		return string(input[file.Offset(expression.Pos()):file.Offset(expression.End())])
	}
	var name bytes.Buffer
	config := printer.Config{Mode: printer.UseSpaces | printer.TabIndent | printer.CallName, Tabwidth: 8}
	if err := config.Fprint(&name, fset, expression); err != nil {
		panic(err)
	}
	return name.String()
}
