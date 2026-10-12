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

// expressionResult keeps evaluation steps with the values they produce.
type expressionResult struct {
	statements []ast.Stmt
	values     []expressionValue
}

func (result expressionResult) expressions() []ast.Expr {
	if result.values == nil {
		return nil
	}
	expressions := make([]ast.Expr, len(result.values))
	for i, value := range result.values {
		expressions[i] = value.expression
	}
	return expressions
}

func (r *rewrite) result(statements []ast.Stmt, expressions []ast.Expr) expressionResult {
	result := expressionResult{statements: statements}
	if expressions != nil {
		result.values = make([]expressionValue, len(expressions))
		for i, expression := range expressions {
			result.values[i] = r.value(expression)
		}
	}
	return result
}
