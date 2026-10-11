package lowering

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
)

func integer(value int) ast.Expr { return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(value)} }

func (r *rewrite) rangeStatement(s *ast.RangeStmt) []ast.Stmt {
	copy := *s
	if value := r.source.Package.TypesInfo.Types[s.X].Value; value != nil && value.Kind() == constant.Int && constant.Sign(value) <= 0 && r.contains(s) {
		suppressed := r.suppressed
		r.suppressed = true
		copy.Body = r.block(s.Body)
		r.suppressed = suppressed
		// No iteration can execute the assignment operands or body.
		for expression := range r.propagations {
			if contains(s, expression) {
				r.lowered[expression] = true
			}
		}
		r.inserted = true
		return []ast.Stmt{&copy}
	}
	copy.Body = r.block(s.Body)
	before, refs := r.expression(s.X)
	copy.X = refs[0]
	if s.Tok == token.ASSIGN && (r.contains(s.Key) || r.contains(s.Value)) {
		if value := r.source.Package.TypesInfo.Types[s.X].Value; value != nil && value.Kind() == constant.Int {
			return r.integerRange(s, &copy, before)
		}
		var stores []ast.Stmt
		var lhs, rhs []ast.Expr
		for _, pair := range []struct {
			original ast.Expr
			output   *ast.Expr
		}{{s.Key, &copy.Key}, {s.Value, &copy.Value}} {
			if pair.original == nil {
				continue
			}
			value := r.name()
			*pair.output = value
			before = append(before, variable(value, r.temporaryType(r.typeOf(pair.original), pair.original), nil))
			setup, place := r.place(pair.original)
			stores = append(stores, setup...)
			lhs = append(lhs, place)
			rhs = append(rhs, value)
		}
		copy.Tok = token.ASSIGN
		stores = append(stores, r.orderedStores(lhs, rhs, token.ASSIGN)...)
		copy.Body = &ast.BlockStmt{List: append(stores, copy.Body)}
	}
	return append(before, &copy)
}

func (r *rewrite) switchStatement(s *ast.SwitchStmt) []ast.Stmt {
	copy := *s
	var init []ast.Stmt
	if s.Init != nil {
		init = r.header(s.Init)
	}
	tagBefore, tag := r.expression(s.Tag)
	var cases []*ast.CaseClause
	changedCases := false
	for _, statement := range s.Body.List {
		clause := statement.(*ast.CaseClause)
		for _, expr := range clause.List {
			if r.contains(expr) {
				changedCases = true
			}
		}
		c := *clause
		c.Body = nil
		for _, body := range clause.Body {
			c.Body = append(c.Body, r.statement(body)...)
		}
		cases = append(cases, &c)
	}
	copy.Body = &ast.BlockStmt{}
	if !changedCases {
		if len(tag) != 0 {
			copy.Tag = tag[0]
		}
		for _, clause := range cases {
			copy.Body.List = append(copy.Body.List, clause)
		}
		if len(tagBefore) == 0 && len(init) <= 1 {
			if len(init) != 0 {
				copy.Init = init[0]
			}
			return []ast.Stmt{&copy}
		}
		copy.Init = nil
		return []ast.Stmt{&ast.BlockStmt{List: append(append(init, tagBefore...), &copy)}}
	}
	setup := append(init, tagBefore...)
	var tagValue ast.Expr
	if len(tag) != 0 {
		tagValue = tag[0]
		if tv := r.source.Package.TypesInfo.Types[s.Tag]; tv.Value == nil {
			saved, values := r.save(tagValue)
			setup = append(setup, saved...)
			tagValue = values[0]
		}
	}
	selected := r.name()
	setup = append(setup, variable(selected, r.temporaryType(types.Typ[types.Int], firstExpression(s)), integer(-1)))
	defaultCase := -1
	for i, clause := range cases {
		if len(clause.List) == 0 {
			defaultCase = i
		}
		for _, expr := range clause.List {
			before, values := r.expression(expr)
			condition := values[0]
			if tagValue != nil {
				condition = &ast.BinaryExpr{X: tagValue, Op: token.EQL, Y: condition}
			}
			body := append(before, &ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{selected}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(i)}}}}})
			setup = append(setup, &ast.IfStmt{Cond: &ast.BinaryExpr{X: selected, Op: token.EQL, Y: integer(-1)}, Body: &ast.BlockStmt{List: body}})
		}
		clause.List = []ast.Expr{integer(i)}
		copy.Body.List = append(copy.Body.List, clause)
	}
	if defaultCase >= 0 {
		setup = append(setup, &ast.IfStmt{Cond: &ast.BinaryExpr{X: selected, Op: token.EQL, Y: integer(-1)}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{selected}, Tok: token.ASSIGN, Rhs: []ast.Expr{integer(defaultCase)}}}}})
	}
	copy.Init = nil
	copy.Tag = selected
	return []ast.Stmt{&ast.BlockStmt{List: append(setup, &copy)}}
}

func (r *rewrite) typeSwitchStatement(s *ast.TypeSwitchStmt) []ast.Stmt {
	copy := *s
	var before []ast.Stmt
	if s.Init != nil {
		before = r.header(s.Init)
	}
	var assertion *ast.TypeAssertExpr
	switch assign := s.Assign.(type) {
	case *ast.ExprStmt:
		assertion = assign.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		assertion = assign.Rhs[0].(*ast.TypeAssertExpr)
	}
	setup, values := r.expression(assertion.X)
	before = append(before, setup...)
	a := *assertion
	a.X = values[0]
	switch assign := s.Assign.(type) {
	case *ast.ExprStmt:
		c := *assign
		c.X = &a
		copy.Assign = &c
	case *ast.AssignStmt:
		c := *assign
		c.Rhs = []ast.Expr{&a}
		copy.Assign = &c
	}
	copy.Body = &ast.BlockStmt{}
	for _, statement := range s.Body.List {
		clause := *statement.(*ast.CaseClause)
		clause.Body = nil
		for _, body := range statement.(*ast.CaseClause).Body {
			clause.Body = append(clause.Body, r.statement(body)...)
		}
		copy.Body.List = append(copy.Body.List, &clause)
	}
	if len(before) != 0 {
		copy.Init = nil
		return []ast.Stmt{&ast.BlockStmt{List: append(before, &copy)}}
	}
	return []ast.Stmt{&copy}
}

func firstExpression(node ast.Node) ast.Expr {
	var result ast.Expr
	ast.Inspect(node, func(n ast.Node) bool {
		if result != nil {
			return false
		}
		if expr, ok := n.(ast.Expr); ok {
			result = expr
			return false
		}
		return true
	})
	return result
}
