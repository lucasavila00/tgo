package compiler

import (
	"go/ast"
	"go/types"
)

// functionNames collects names that generated locals must not capture.
func functionNames(body *ast.BlockStmt, signature *types.Signature) map[string]bool {
	names := make(map[string]bool)
	addTupleNames(names, signature.Params())
	addTupleNames(names, signature.Results())
	if receiver := signature.Recv(); receiver != nil && receiver.Name() != "" {
		names[receiver.Name()] = true
	}
	ast.Inspect(body, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			names[identifier.Name] = true
		}
		return true
	})
	return names
}

func addTupleNames(names map[string]bool, tuple *types.Tuple) {
	for index := range tuple.Len() {
		if name := tuple.At(index).Name(); name != "" {
			names[name] = true
		}
	}
}

// freshName returns a short local name without changing source name resolution.
func (l *propagationLowerer) freshName(preferred string) *ast.Ident {
	return ast.NewIdent(freshIdentifier(preferred, l.names))
}
