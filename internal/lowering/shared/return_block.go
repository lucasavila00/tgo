package shared

import (
	"go/ast"
	"go/token"
	"reflect"
)

// returnBlock shares one final return when source expressions always leave the function.
func (r *Context) returnBlock(typ *ast.FuncType, body *ast.BlockStmt) *ast.BlockStmt {
	exit := r.Name()
	var prefix []ast.Stmt
	var values []ast.Expr
	var named []ast.Expr
	for _, field := range typ.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			name := r.Name()
			prefix = append(prefix, Variable(name,
				r.temporaryTypeAt(r.input.Inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type,
					typ.End()), nil))
			values = append(values, name)
			value := ast.Expr(name)
			if len(field.Names) > 0 && field.Names[i].Name != "_" {
				value = ast.NewIdent(field.Names[i].Name)
			}
			named = append(named, value)
		}
	}
	replaced := false
	inner := rewriteReturns(body, func(statement *ast.ReturnStmt) ast.Stmt {
		replaced = true
		results := statement.Results
		if len(results) == 0 {
			results = named
		}
		return &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{Lhs: values, Tok: token.ASSIGN, Rhs: results},
			&ast.BranchStmt{Tok: token.GOTO, Label: exit},
		}}
	}).(*ast.BlockStmt)
	final := ast.Stmt(&ast.ReturnStmt{Results: values})
	if replaced {
		final = &ast.LabeledStmt{Label: exit, Stmt: final}
	}
	return &ast.BlockStmt{List: append(prefix, inner, final)}
}

// rewriteReturns leaves nested function returns in their own function.
func rewriteReturns(node ast.Node, replace func(*ast.ReturnStmt) ast.Stmt) ast.Node {
	if node == nil {
		return nil
	}
	if statement, ok := node.(*ast.ReturnStmt); ok {
		return replace(statement)
	}
	if _, function := node.(*ast.FuncLit); function {
		return node
	}
	original := reflect.ValueOf(node).Elem()
	rewrittenNode := reflect.New(original.Type())
	rewrittenNode.Elem().Set(original)
	changed := rewriteChildFields(
		original,
		rewrittenNode.Elem(),
		func(child ast.Node) ast.Node { return rewriteReturns(child, replace) },
	)
	if !changed {
		return node
	}
	return rewrittenNode.Interface().(ast.Node)
}
