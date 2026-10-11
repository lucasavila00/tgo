package driver

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/lucasavila00/tgo/internal/frontend"
	"golang.org/x/tools/go/packages"
)

func checkPropagationReturns(pkg *packages.Package, operations map[string]frontend.Marker) {
	for _, file := range pkg.Syntax {
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			stack = append(stack, node)
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			operation, ok := operations[name.Name]
			if !ok {
				return true
			}
			var signature *types.Signature
			for i := len(stack) - 1; i >= 0; i-- {
				switch function := stack[i].(type) {
				case *ast.FuncDecl:
					signature, _ = pkg.TypesInfo.Defs[function.Name].Type().(*types.Signature)
				case *ast.FuncLit:
					signature, _ = pkg.TypesInfo.TypeOf(function).(*types.Signature)
				}
				if signature != nil {
					break
				}
			}
			if signature == nil || signature.Results().Len() == 0 {
				return true
			}
			returned := pkg.TypesInfo.TypeOf(call.Args[0])
			if tuple, ok := returned.(*types.Tuple); ok {
				returned = tuple.At(tuple.Len() - 1).Type()
			}
			operator := "!"
			if operation.Wrap {
				operator = "!!"
				returned = types.Universe.Lookup("error").Type()
			}
			target := signature.Results().At(signature.Results().Len() - 1).Type()
			if !types.AssignableTo(returned, target) {
				panic(fmt.Sprintf("%s: %s cannot return error of type %s as %s", operation.Position, operator, returned, target))
			}
			return true
		})
	}
}
