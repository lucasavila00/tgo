package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) selectStatement(s *ast.SelectStmt) []ast.Stmt {
	var roots []ast.Node
	for _, stmt := range s.Body.List {
		roots = append(roots, stmt.(*ast.CommClause).Comm)
	}
	exit := r.prepareBreakExit(s, roots...)
	selectStatement := *s
	selectStatement.Body = &ast.BlockStmt{}
	var before []ast.Stmt
	commChanged := false
	for _, statement := range s.Body.List {
		if r.contains(statement.(*ast.CommClause).Comm) {
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
				sendStatement := *comm
				channelSetup, savedChannel := r.capture(comm.Chan)
				before = append(before, channelSetup...)
				valueSetup, sentValue := r.capture(comm.Value)
				before = append(before, valueSetup...)
				sendStatement.Chan = savedChannel
				sendStatement.Value = sentValue
				clause.Comm = &sendStatement
			case *ast.ExprStmt:
				receive := ast.Unparen(comm.X).(*ast.UnaryExpr)
				receiveExpression := *receive
				channelSetup, savedChannel := r.capture(receive.X)
				before = append(before, channelSetup...)
				receiveExpression.X = savedChannel
				if receive == ast.Unparen(r.target) {
					r.inserted = true
					value := r.name()
					before = append(before, variable(value, r.temporaryType(r.typeOf(receive), receive), nil))
					clause.Comm = &ast.AssignStmt{Lhs: []ast.Expr{value}, Tok: token.ASSIGN, Rhs: []ast.Expr{&receiveExpression}}
					body = append(body, r.after([]ast.Expr{value})...)
					body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
				} else {
					clause.Comm = &ast.ExprStmt{X: &receiveExpression}
				}
			case *ast.AssignStmt:
				receive := ast.Unparen(comm.Rhs[0]).(*ast.UnaryExpr)
				receiveExpression := *receive
				channelSetup, savedChannel := r.capture(receive.X)
				before = append(before, channelSetup...)
				receiveExpression.X = savedChannel
				values := make([]ast.Expr, len(comm.Lhs))
				for i := range values {
					values[i] = r.name()
					typ := r.typeOf(receive)
					if tuple, ok := typ.(*types.Tuple); ok {
						typ = tuple.At(i).Type()
					} else if i == 1 {
						typ = types.Typ[types.Bool]
					}
					before = append(before, variable(values[i].(*ast.Ident), r.temporaryType(typ, receive), nil))
				}
				clause.Comm = &ast.AssignStmt{Lhs: values, Tok: token.ASSIGN, Rhs: []ast.Expr{&receiveExpression}}
				if receive == ast.Unparen(r.target) {
					r.inserted = true
					body = append(body, r.after(values)...)
				}
				store := *comm
				store.Rhs = values
				if comm.Tok == token.ASSIGN {
					store.Lhs = make([]ast.Expr, len(comm.Lhs))
					for i, lhs := range comm.Lhs {
						placeSetup, destination := r.place(lhs)
						body = append(body, placeSetup...)
						store.Lhs[i] = destination
					}
				}
				deferred := false
				for _, place := range store.Lhs {
					deferred = deferred || r.deferredPlace(place)
				}
				if deferred {
					body = append(body, r.orderedStores(store.Lhs, store.Rhs, store.Tok)...)
				} else {
					body = append(body, &store)
				}
			}
		}
		for _, statement := range original.Body {
			body = append(body, r.statement(statement)...)
		}
		clause.Body = body
		selectStatement.Body.List = append(selectStatement.Body.List, &clause)
	}
	return r.withBreakExit(s, &selectStatement, append(before, &selectStatement), exit)
}
