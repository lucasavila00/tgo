package compilerv2

import (
	"go/ast"
	"go/token"
)

func (r *rewrite) selectStatement(s *ast.SelectStmt) []ast.Stmt {
	copy := *s
	copy.Body = &ast.BlockStmt{}
	var before []ast.Stmt
	commChanged := false
	for _, statement := range s.Body.List {
		if contains(statement.(*ast.CommClause).Comm, r.target) {
			commChanged = true
		}
	}
	for _, statement := range s.Body.List {
		original := statement.(*ast.CommClause)
		clause := *original
		var body []ast.Stmt
		if commChanged && original.Comm != nil {
			switch comm := original.Comm.(type) {
			case *ast.SendStmt:
				c := *comm
				setup, channel := r.capture(comm.Chan)
				before = append(before, setup...)
				setup, value := r.capture(comm.Value)
				before = append(before, setup...)
				c.Chan, c.Value = channel, value
				clause.Comm = &c
			case *ast.ExprStmt:
				receive := comm.X.(*ast.UnaryExpr)
				c := *receive
				setup, channel := r.capture(receive.X)
				before = append(before, setup...)
				c.X = channel
				if receive == r.target {
					r.inserted = true
					value := r.name()
					clause.Comm = &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.DEFINE, Rhs: []ast.Expr{&c}}
					body = append(body, r.after([]ast.Expr{value})...)
					body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
				} else {
					clause.Comm = &ast.ExprStmt{X: &c}
				}
			case *ast.AssignStmt:
				receive := comm.Rhs[0].(*ast.UnaryExpr)
				c := *receive
				setup, channel := r.capture(receive.X)
				before = append(before, setup...)
				c.X = channel
				values := make([]ast.Expr, len(comm.Lhs))
				for i := range values {
					values[i] = r.name()
				}
				clause.Comm = &ast.AssignStmt{Lhs: values, Tok: token.DEFINE, Rhs: []ast.Expr{&c}}
				if receive == r.target {
					r.inserted = true
					body = append(body, r.after(values)...)
				}
				store := *comm
				store.Rhs = values
				if comm.Tok == token.ASSIGN {
					store.Lhs = make([]ast.Expr, len(comm.Lhs))
					for i, lhs := range comm.Lhs {
						setup, place := r.place(lhs)
						body = append(body, setup...)
						store.Lhs[i] = place
					}
				}
				body = append(body, &store)
			}
		}
		for _, statement := range original.Body {
			body = append(body, r.statement(statement)...)
		}
		clause.Body = body
		copy.Body.List = append(copy.Body.List, &clause)
	}
	return append(before, &copy)
}
