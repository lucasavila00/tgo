package lowering

import (
	"go/ast"
	"go/token"
	"reflect"
)

// returnBlock shares one final return when source expressions always leave the function.
func (r *rewrite) returnBlock(typ *ast.FuncType, body *ast.BlockStmt) *ast.BlockStmt {
	exit := r.name()
	var prefix []ast.Stmt
	var values []ast.Expr
	var named []ast.Expr
	for _, field := range typ.Results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			name := r.name()
			prefix = append(prefix, variable(name,
				r.temporaryTypeAt(r.input.source.inventory.Checked.TypesInfo.TypeOf(field.Type), field.Type,
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
	changed := false
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
		output := rewrittenNode.Elem().Field(i)
		if child, ok := field.Interface().(ast.Node); ok {
			if reflect.ValueOf(child).IsNil() {
				continue
			}
			rewritten := rewriteReturns(child, replace)
			if rewritten != child {
				changed = true
				output.Set(reflect.ValueOf(rewritten))
			}
		} else if field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				child, ok := field.Index(j).Interface().(ast.Node)
				if !ok || reflect.ValueOf(child).IsNil() {
					continue
				}
				rewritten := rewriteReturns(child, replace)
				if rewritten != child {
					changed = true
					if output.Pointer() == field.Pointer() {
						output.Set(reflect.MakeSlice(field.Type(), field.Len(), field.Len()))
						reflect.Copy(output, field)
					}
					output.Index(j).Set(reflect.ValueOf(rewritten))
				}
			}
		}
	}
	if !changed {
		return node
	}
	return rewrittenNode.Interface().(ast.Node)
}
