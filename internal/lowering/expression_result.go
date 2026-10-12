package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// expressionValue keeps a Go expression with its type and source expression.
// A tuple type means this expression supplies multiple results, such as a call.
type expressionValue struct {
	expression ast.Expr
	typ        types.Type
	original   ast.Expr
	temporary  bool
}

func (r *rewrite) value(expression ast.Expr) expressionValue {
	if value, ok := r.functionState.values[expression]; ok {
		return value
	}
	return expressionValue{
		expression: expression,
		typ:        r.input.source.inventory.Checked.TypesInfo.TypeOf(expression),
		original:   expression,
	}
}
