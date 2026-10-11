package driver

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/lucasavila00/tgo/internal/frontend"
	"golang.org/x/tools/go/packages"
)

func rejectUnsafePropagation(pkg *packages.Package, operations map[string]frontend.Marker) {
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
					operation, ok := operations[name.Name]
					if !ok {
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
