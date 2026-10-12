package shared

import (
	"go/ast"
)

// ValueOf returns the checked type and source identity for an expression.
func (r *Context) ValueOf(expression ast.Expr) Value {
	if value, ok := r.functionState.values[expression]; ok {
		return value
	}
	return Value{
		Expression: expression,
		Type:       r.Input.Inventory.Checked.TypesInfo.TypeOf(expression),
		Original:   expression,
	}
}

// ResultOf joins statements with their typed output expressions.
func (r *Context) ResultOf(statements []ast.Stmt, expressions []ast.Expr) Result {
	result := Result{Statements: statements}
	if expressions != nil {
		result.Values = make([]Value, len(expressions))
		for i, expression := range expressions {
			result.Values[i] = r.ValueOf(expression)
		}
	}
	return result
}
