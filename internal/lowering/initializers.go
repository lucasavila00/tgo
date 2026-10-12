package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) packageConditional(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok &&
			(r.input.conditionals[call] != nil || r.input.comprehensions[call] != nil) {
			found = true
			return false
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
	return found
}

func (r *rewrite) packageInitializer(expression ast.Expr) ast.Expr {
	initializer := &rewrite{input: r.input, request: r.request, file: r.file, functionState: &functionState{}}
	typ := initializer.typeOf(expression)
	results := &ast.FieldList{}
	if tuple, ok := typ.(*types.Tuple); ok {
		for i := 0; i < tuple.Len(); i++ {
			results.List = append(results.List, &ast.Field{Type: initializer.temporaryType(tuple.At(i).Type(), expression)})
		}
	} else {
		results.List = append(results.List, &ast.Field{Type: initializer.temporaryType(typ, expression)})
	}
	before, values := initializer.expression(expression)
	body := &ast.BlockStmt{List: append(before, &ast.ReturnStmt{Results: values})}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: results}, Body: body}}
}
