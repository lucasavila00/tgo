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
			builtin := unsafeLayoutBuiltin(pkg, call)
			if builtin == nil {
				return true
			}
			for _, operand := range call.Args {
				rejectOperandPropagation(pkg, operand, builtin.Name())
			}
			return true
		})
	}
}

func unsafeLayoutBuiltin(pkg *checker.Package, call *ast.CallExpr) *types.Builtin {
	var id *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = fun
	case *ast.SelectorExpr:
		id = fun.Sel
	}
	builtin, ok := pkg.TypesInfo.Uses[id].(*types.Builtin)
	if !ok || builtin.Pkg() == nil || builtin.Pkg().Path() != "unsafe" {
		return nil
	}
	switch builtin.Name() {
	case "Sizeof", "Alignof", "Offsetof":
	default:
		return nil
	}
	return builtin
}

func rejectOperandPropagation(pkg *checker.Package, operand ast.Expr, builtinName string) {
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
		if !ok || operation.Conditional || operation.Comprehension || operation.Element || operation.ShortReturn {
			return true
		}
		operator := "!"
		if operation.Wrap {
			operator = "!!"
		}
		panic(fmt.Sprintf("%s: %s is not allowed inside unsafe.%s operands", operation.Position, operator, builtinName))
	})
}
