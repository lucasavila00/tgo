package compiler

import (
	"go/ast"
	"go/types"
)

func comprehensionMarker(
	source *source,
	expression ast.Expr,
) (comprehensionSource, *ast.FuncLit, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return comprehensionSource{}, nil, false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return comprehensionSource{}, nil, false
	}
	metadata, ok := source.Comprehensions[name.Name]
	if !ok {
		return comprehensionSource{}, nil, false
	}
	function, ok := call.Args[0].(*ast.FuncLit)
	return metadata, function, ok
}

func underlyingSlice(value types.Type) *types.Slice {
	value = types.Unalias(value)
	if named, ok := value.(*types.Named); ok {
		value = named.Underlying()
	}
	slice, _ := value.(*types.Slice)
	return slice
}

func comprehensionTerminal(body *ast.BlockStmt) *ast.BlockStmt {
	for len(body.List) == 1 {
		switch statement := body.List[0].(type) {
		case *ast.ForStmt:
			body = statement.Body
		case *ast.RangeStmt:
			body = statement.Body
		case *ast.IfStmt:
			if statement.Else != nil {
				return body
			}
			body = statement.Body
		default:
			return body
		}
	}
	return body
}
