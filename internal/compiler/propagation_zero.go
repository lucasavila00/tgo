package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// zeroExpression returns a direct Go zero when its spelling is simple and exact.
func (l *propagationLowerer) zeroExpression(
	value types.Type,
	typeExpression ast.Expr,
	position token.Pos,
) (ast.Expr, bool) {
	underlying := types.Unalias(value)
	if named, ok := underlying.(*types.Named); ok {
		underlying = named.Underlying()
	}
	switch item := underlying.(type) {
	case *types.Basic:
		switch {
		case item.Info()&types.IsBoolean != 0:
			return l.unit.generatedUniverse("false", position), true
		case item.Info()&types.IsString != 0:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}, true
		case item.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}, true
		case item.Kind() == types.UnsafePointer || item.Kind() == types.UntypedNil:
			return l.unit.generatedUniverse("nil", position), true
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan,
		*types.Signature, *types.Interface:
		return l.unit.generatedUniverse("nil", position), true
	case *types.Array, *types.Struct:
		return &ast.CompositeLit{Type: typeExpression}, true
	}
	return nil, false
}
