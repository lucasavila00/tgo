package shared

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *Context) packageConditional(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok &&
			(r.Input.Conditionals[call] != nil || r.Input.Comprehensions[call] != nil) {
			found = true
			return false
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
	return found
}

func (r *Context) packageInitializer(expression ast.Expr) ast.Expr {
	initializer := &Context{Input: r.Input, Request: r.Request, file: r.file, functionState: &functionState{}}
	typ := initializer.TypeOf(expression)
	results := &ast.FieldList{}
	if tuple, ok := typ.(*types.Tuple); ok {
		for i := 0; i < tuple.Len(); i++ {
			results.List = append(results.List, &ast.Field{Type: initializer.TemporaryType(tuple.At(i).Type(), expression)})
		}
	} else {
		results.List = append(results.List, &ast.Field{Type: initializer.TemporaryType(typ, expression)})
	}
	beforeEvaluation := initializer.Expression(expression)
	before, values := beforeEvaluation.Statements, beforeEvaluation.Expressions()
	body := &ast.BlockStmt{List: append(before, &ast.ReturnStmt{Results: values})}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: results}, Body: body}}
}
