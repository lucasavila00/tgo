package driver

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker"
)

func rejectUnsafePropagation(pkg *checker.Package) {
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			var id *ast.Ident
			switch fun := ast.Unparen(call.Fun).(type) {
			case *ast.Ident:
				id = fun
			case *ast.SelectorExpr:
				id = fun.Sel
			}
			builtin, ok := pkg.TypesInfo.Uses[id].(*types.Builtin)
			if !ok || builtin.Pkg() == nil || builtin.Pkg().Path() != "unsafe" {
				return true
			}
			switch builtin.Name() {
			case "Sizeof", "Alignof", "Offsetof":
			default:
				return true
			}
			for _, operand := range call.Args {
				ast.Inspect(operand, func(node ast.Node) bool {
					marker, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					name, ok := marker.Fun.(*ast.Ident)
					if !ok {
						return true
					}
					operation, ok := pkg.Operations[name.Name]
					if !ok || operation.Conditional {
						return true
					}
					operator := "!"
					if operation.Wrap {
						operator = "!!"
					}
					panic(fmt.Sprintf("%s: %s is not allowed inside unsafe.%s operands", operation.Position, operator, builtin.Name()))
				})
			}
			return true
		})
	}
}
