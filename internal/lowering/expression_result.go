package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/lowering/shared"
)

type expressionValue = shared.Value
type expressionResult = shared.Result

func (r *rewrite) value(expression ast.Expr) expressionValue {
	if value, ok := r.functionState.values[expression]; ok {
		return value
	}
	return expressionValue{
		Expression: expression,
		Type:       r.input.source.inventory.Checked.TypesInfo.TypeOf(expression),
		Original:   expression,
	}
}

func (r *rewrite) result(statements []ast.Stmt, expressions []ast.Expr) expressionResult {
	result := expressionResult{Statements: statements}
	if expressions != nil {
		result.Values = make([]expressionValue, len(expressions))
		for i, expression := range expressions {
			result.Values[i] = r.value(expression)
		}
	}
	return result
}
