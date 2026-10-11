package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) packageConditional(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && (r.conditionals[call] != nil || r.comprehensions[call] != nil) {
			found = true
			return false
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
	return found
}

func (r *rewrite) packageInitializer(expression ast.Expr) ast.Expr {
	typ := r.typeOf(expression)
	results := &ast.FieldList{}
	if tuple, ok := typ.(*types.Tuple); ok {
		for i := 0; i < tuple.Len(); i++ {
			results.List = append(results.List, &ast.Field{Type: r.temporaryType(tuple.At(i).Type(), expression)})
		}
	} else {
		results.List = append(results.List, &ast.Field{Type: r.temporaryType(typ, expression)})
	}
	before, values := r.expression(expression)
	body := &ast.BlockStmt{List: append(before, &ast.ReturnStmt{Results: values})}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: results}, Body: body}}
}

func (r *rewrite) conditionalCall(call *ast.CallExpr, statement ast.Stmt) ([]ast.Stmt, []ast.Expr) {
	result := r.name()
	typ := types.Default(r.source.checked.TypesInfo.TypeOf(call))
	declaration := variable(result, r.temporaryType(typ, call), nil)
	if r.temporaryTypes == nil {
		r.temporaryTypes = make(map[ast.Expr]types.Type)
	}
	if r.origins == nil {
		r.origins = make(map[ast.Expr]ast.Expr)
	}
	r.temporaryTypes[result] = typ
	r.origins[result] = call
	body := r.valueStatement(statement, result)
	r.lowered[call] = true
	return append([]ast.Stmt{declaration}, body...), []ast.Expr{result}
}

func (r *rewrite) conditionalStatement(statement *ast.IfStmt, result *ast.Ident) []ast.Stmt {
	copy := *statement
	copy.Body = r.conditionalBlock(statement.Body, result)
	switch alternative := statement.Else.(type) {
	case *ast.BlockStmt:
		copy.Else = r.conditionalBlock(alternative, result)
	case *ast.IfStmt:
		output := r.conditionalStatement(alternative, result)
		if len(output) == 1 {
			copy.Else = output[0]
		} else {
			copy.Else = &ast.BlockStmt{List: output}
		}
	}
	var setup []ast.Stmt
	if statement.Init != nil {
		setup = r.header(statement.Init)
	}
	before, condition := r.expression(statement.Cond)
	copy.Cond = condition[0]
	if len(before) == 0 && len(setup) <= 1 {
		if len(setup) != 0 {
			copy.Init = setup[0]
		}
		return []ast.Stmt{&copy}
	}
	copy.Init = nil
	return []ast.Stmt{&ast.BlockStmt{List: append(append(setup, before...), &copy)}}
}

func (r *rewrite) conditionalBlock(body *ast.BlockStmt, result *ast.Ident) *ast.BlockStmt {
	copy := *body
	copy.List = nil
	if completes, recorded := r.source.checked.TypesInfo.TGoBranches[body]; recorded && !completes {
		result = nil
	}
	last := len(body.List) - 1
	for last >= 0 {
		if _, empty := body.List[last].(*ast.EmptyStmt); !empty {
			break
		}
		last--
	}
	for i, statement := range body.List {
		if i == last {
			copy.List = append(copy.List, r.conditionalTail(statement, result)...)
		} else {
			copy.List = append(copy.List, r.statement(statement)...)
		}
	}
	return &copy
}

func (r *rewrite) conditionalTail(statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.LabeledStmt:
		tail := statement.Stmt
		for {
			label, ok := tail.(*ast.LabeledStmt)
			if !ok {
				break
			}
			tail = label.Stmt
		}
		switch tail.(type) {
		case *ast.ExprStmt, *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		default:
			return r.statement(statement)
		}
		copy := *statement
		output := r.conditionalTail(statement.Stmt, result)
		if len(output) == 1 {
			copy.Stmt = output[0]
		} else {
			copy.Stmt = &ast.BlockStmt{List: output}
		}
		return []ast.Stmt{&copy}
	case *ast.IfStmt:
		return r.conditionalStatement(statement, result)
	case *ast.SwitchStmt:
		return r.valueSwitchStatement(statement, result)
	case *ast.TypeSwitchStmt:
		return r.valueTypeSwitchStatement(statement, result)
	case *ast.ExprStmt:
		if tv := r.source.checked.TypesInfo.Types[statement.X]; !tv.IsVoid() {
			before, values := r.expression(statement.X)
			if result == nil {
				for _, value := range values {
					if tv.Value != nil {
						value = &ast.BinaryExpr{X: value, Op: token.EQL, Y: value}
					}
					if tv.IsNil() {
						value = &ast.CallExpr{Fun: &ast.InterfaceType{Methods: &ast.FieldList{}}, Args: []ast.Expr{value}}
					}
					before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
				}
				return before
			}
			return append(before, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: values})
		}
	}
	return r.statement(statement)
}

func (r *rewrite) valueStatement(statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.IfStmt:
		return r.conditionalStatement(statement, result)
	case *ast.SwitchStmt:
		return r.valueSwitchStatement(statement, result)
	case *ast.TypeSwitchStmt:
		return r.valueTypeSwitchStatement(statement, result)
	}
	return nil
}
