package compilerv2

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

type rewrite struct {
	source *Source
	target ast.Expr
	after  After
	names  map[string]bool
	next   int
}

func (r *rewrite) name() *ast.Ident {
	for {
		r.next++
		name := fmt.Sprintf("tgo%d", r.next)
		if !r.names[name] {
			r.names[name] = true
			return ast.NewIdent(name)
		}
	}
}

func (r *rewrite) save(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	count := 1
	if tuple, ok := r.source.Package.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
		count = tuple.Len()
	}
	if count == 0 {
		return []ast.Stmt{&ast.ExprStmt{X: expr}}, nil
	}
	refs := make([]ast.Expr, count)
	for i := range refs {
		refs[i] = r.name()
	}
	return []ast.Stmt{&ast.AssignStmt{Lhs: refs, Tok: token.DEFINE, Rhs: []ast.Expr{expr}}}, refs
}

// operands saves earlier values before a later operand emits statements.
func (r *rewrite) operands(input []ast.Expr) ([]ast.Stmt, []ast.Expr) {
	var statements []ast.Stmt
	var values []ast.Expr
	for _, expr := range input {
		before, refs := r.expression(expr)
		if len(before) != 0 {
			for i, value := range values {
				if tv, ok := r.source.Package.TypesInfo.Types[value]; ok && (tv.Value != nil || tv.IsType() || tv.IsBuiltin()) {
					continue
				}
				if _, ok := value.(*ast.Ident); ok && r.source.Package.TypesInfo.TypeOf(value) == nil {
					continue
				}
				save, saved := r.save(value)
				statements = append(statements, save...)
				values[i] = saved[0]
			}
			statements = append(statements, before...)
		}
		values = append(values, refs...)
	}
	return statements, values
}

func (r *rewrite) expression(expr ast.Expr) ([]ast.Stmt, []ast.Expr) {
	if expr == nil {
		return nil, nil
	}
	var before []ast.Stmt
	var result ast.Expr = expr
	switch e := expr.(type) {
	case *ast.CallExpr:
		copy := *e
		operands := append([]ast.Expr{e.Fun}, e.Args...)
		var values []ast.Expr
		before, values = r.operands(operands)
		copy.Fun, copy.Args = values[0], values[1:]
		result = &copy
	case *ast.ParenExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		if len(values) == 1 {
			copy.X = values[0]
			result = &copy
		} else {
			return before, values
		}
	case *ast.UnaryExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.BinaryExpr:
		copy := *e
		if e.Op == token.LAND || e.Op == token.LOR {
			leftBefore, left := r.expression(e.X)
			rightBefore, right := r.expression(e.Y)
			before = leftBefore
			copy.X, copy.Y = left[0], right[0]
			if len(rightBefore) == 0 {
				result = &copy
				break
			}
			value := r.name()
			// The false operand gives the local the original result type without evaluating e.
			zero := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: token.NEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.BinaryExpr{X: zero, Op: token.LAND, Y: e}}})
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: left})
			var condition ast.Expr = value
			if e.Op == token.LOR {
				condition = &ast.UnaryExpr{Op: token.NOT, X: value}
			}
			rightBefore = append(rightBefore, &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: right})
			before = append(before, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: rightBefore}})
			result = value
		} else {
			var values []ast.Expr
			before, values = r.operands([]ast.Expr{e.X, e.Y})
			copy.X, copy.Y = values[0], values[1]
			result = &copy
		}
	case *ast.SelectorExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.expression(e.X)
		copy.X = values[0]
		result = &copy
	case *ast.IndexExpr:
		copy := *e
		var values []ast.Expr
		before, values = r.operands([]ast.Expr{e.X, e.Index})
		copy.X, copy.Index = values[0], values[1]
		result = &copy
	}
	if expr == r.target {
		// Read types from the original expression, not from the new tree.
		count := 1
		if tuple, ok := r.source.Package.TypesInfo.TypeOf(expr).(*types.Tuple); ok {
			count = tuple.Len()
		}
		refs := make([]ast.Expr, count)
		for i := range refs {
			refs[i] = r.name()
		}
		if count == 0 {
			before = append(before, &ast.ExprStmt{X: result})
		} else {
			before = append(before, &ast.AssignStmt{Lhs: refs, Tok: token.DEFINE, Rhs: []ast.Expr{result}})
		}
		before = append(before, r.after(refs)...)
		return before, refs
	}
	return before, []ast.Expr{result}
}
