package compilerv2

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (r *rewrite) block(body *ast.BlockStmt) *ast.BlockStmt {
	if body == nil || (!contains(body, r.target) && !r.hasMappedBranch(body)) {
		return body
	}
	copy := *body
	copy.List = nil
	for _, statement := range body.List {
		copy.List = append(copy.List, r.statement(statement)...)
	}
	return &copy
}

func (r *rewrite) statement(statement ast.Stmt) []ast.Stmt {
	if branch, ok := statement.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO {
		if name := r.labels[branch.Label.Name]; name != "" {
			copy := *branch
			copy.Label = ast.NewIdent(name)
			return []ast.Stmt{&copy}
		}
	}
	if !contains(statement, r.target) && !r.hasMappedBranch(statement) {
		return []ast.Stmt{statement}
	}
	output := r.statementInner(statement)
	if len(output) <= 1 {
		return output
	}
	switch s := statement.(type) {
	case *ast.DeclStmt, *ast.LabeledStmt:
		return output
	case *ast.AssignStmt:
		if s.Tok == token.DEFINE {
			return output
		}
	}
	return []ast.Stmt{&ast.BlockStmt{List: output}}
}

func (r *rewrite) statementInner(statement ast.Stmt) []ast.Stmt {
	switch s := statement.(type) {
	case *ast.BlockStmt:
		return []ast.Stmt{r.block(s)}
	case *ast.RangeStmt:
		return r.rangeStatement(s)
	case *ast.SwitchStmt:
		return r.switchStatement(s)
	case *ast.TypeSwitchStmt:
		return r.typeSwitchStatement(s)
	case *ast.SelectStmt:
		return r.selectStatement(s)
	case *ast.SendStmt:
		copy := *s
		before, values := r.operands([]ast.Expr{s.Chan, s.Value})
		copy.Chan, copy.Value = values[0], values[1]
		return append(before, &copy)
	case *ast.ExprStmt:
		before, refs := r.expression(s.X)
		if len(refs) == 0 {
			if call, ok := s.X.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok {
					if object := r.source.Package.TypesInfo.Uses[id]; object == types.Universe.Lookup("panic") {
						// Keep the terminating statement after unreachable inserted code.
						return append(before, s)
					}
				}
			}
			return before
		}
		copy := *s
		copy.X = refs[0]
		if s.X == r.target {
			// A completed expression statement has no further operation.
			for _, ref := range refs {
				before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ref}})
			}
			return before
		}
		return append(before, &copy)
	case *ast.ReturnStmt:
		copy := *s
		before, values := r.operands(s.Results)
		copy.Results = values
		return append(before, &copy)
	case *ast.AssignStmt:
		copy := *s
		var before []ast.Stmt
		copy.Lhs = make([]ast.Expr, len(s.Lhs))
		if s.Tok == token.DEFINE {
			copy.Lhs = s.Lhs
		} else {
			for i, lhs := range s.Lhs {
				setup, place := r.place(lhs)
				before = append(before, setup...)
				copy.Lhs[i] = place
			}
		}
		setup, values := r.operands(s.Rhs)
		before = append(before, setup...)
		copy.Rhs = values
		return append(before, &copy)
	case *ast.IncDecStmt:
		copy := *s
		before, place := r.place(s.X)
		copy.X = place
		return append(before, &copy)
	case *ast.DeclStmt:
		declaration := s.Decl.(*ast.GenDecl)
		var output []ast.Stmt
		for _, spec := range declaration.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				output = append(output, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: declaration.Tok, Specs: []ast.Spec{spec}}})
				continue
			}
			copy := *value
			before, values := r.operands(value.Values)
			copy.Values = values
			output = append(output, before...)
			output = append(output, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: declaration.Tok, Specs: []ast.Spec{&copy}}})
		}
		return output
	case *ast.IfStmt:
		copy := *s
		copy.Body = r.block(s.Body)
		if s.Else != nil {
			statements := r.statement(s.Else)
			if len(statements) == 1 {
				copy.Else = statements[0]
			} else {
				copy.Else = &ast.BlockStmt{List: statements}
			}
		}
		var setup []ast.Stmt
		if s.Init != nil {
			setup = r.header(s.Init)
		}
		before, refs := r.expression(s.Cond)
		copy.Cond = refs[0]
		if len(before) == 0 && len(setup) <= 1 {
			if len(setup) != 0 {
				copy.Init = setup[0]
			}
			return []ast.Stmt{&copy}
		}
		copy.Init = nil
		return []ast.Stmt{&ast.BlockStmt{List: append(append(setup, before...), &copy)}}
	case *ast.ForStmt:
		copy := *s
		copy.Body = r.block(s.Body)
		condition, refs := r.expression(s.Cond)
		var post []ast.Stmt
		if s.Post != nil {
			post = r.header(s.Post)
		}
		var init []ast.Stmt
		if s.Init != nil {
			init = r.header(s.Init)
		}
		if len(condition) == 0 && len(post) <= 1 && len(init) <= 1 {
			if len(refs) != 0 {
				copy.Cond = refs[0]
			}
			if len(post) != 0 {
				copy.Post = post[0]
			}
			if len(init) != 0 {
				copy.Init = init[0]
			}
			return []ast.Stmt{&copy}
		}
		var outer []ast.Stmt
		if len(init) != 0 {
			outer = append(outer, init[:len(init)-1]...)
			copy.Init = init[len(init)-1]
		}
		var body []ast.Stmt
		if len(post) != 0 {
			first := r.name()
			outer = append(outer, &ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.DEFINE, Rhs: []ast.Expr{boolean(true)}})
			body = append(body, &ast.IfStmt{Cond: first, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.ASSIGN, Rhs: []ast.Expr{boolean(false)}}}}, Else: &ast.BlockStmt{List: post}})
		}
		body = append(body, condition...)
		if len(refs) != 0 {
			body = append(body, &ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: refs[0]}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}}})
		}
		body = append(body, copy.Body)
		copy.Cond, copy.Post = nil, nil
		copy.Body = &ast.BlockStmt{List: body}
		return []ast.Stmt{&ast.BlockStmt{List: append(outer, &copy)}}
	case *ast.GoStmt:
		copy := *s
		before, refs := r.expression(s.Call)
		copy.Call = refs[0].(*ast.CallExpr)
		return append(before, &copy)
	case *ast.DeferStmt:
		copy := *s
		before, refs := r.expression(s.Call)
		copy.Call = refs[0].(*ast.CallExpr)
		return append(before, &copy)
	case *ast.LabeledStmt:
		copy := *s
		jump := r.hasGoto(s)
		var inner *ast.Ident
		if jump && hasLabelBranch(s.Stmt, s.Label.Name) {
			inner = r.name()
			if r.labels == nil {
				r.labels = make(map[string]string)
			}
			r.labels[s.Label.Name] = inner.Name
			defer delete(r.labels, s.Label.Name)
		}
		statements := r.statement(s.Stmt)
		if !jump {
			if len(statements) == 1 {
				if block, ok := statements[0].(*ast.BlockStmt); ok {
					last := len(block.List) - 1
					copy.Stmt = block.List[last]
					block.List[last] = &copy
					return statements
				}
			}
			copy.Stmt = statements[len(statements)-1]
			return append(statements[:len(statements)-1], &copy)
		}
		if inner != nil {
			attachLabel(statements, inner)
		}
		copy.Stmt = &ast.BlockStmt{List: statements}
		return []ast.Stmt{&copy}
	}
	return []ast.Stmt{statement}
}

func attachLabel(statements []ast.Stmt, label *ast.Ident) {
	last := len(statements) - 1
	if block, ok := statements[last].(*ast.BlockStmt); ok {
		attachLabel(block.List, label)
		return
	}
	statements[last] = &ast.LabeledStmt{Label: label, Stmt: statements[last]}
}

func hasLabelBranch(node ast.Node, label string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO && branch.Label.Name == label {
			found = true
		}
		return true
	})
	return found
}

func (r *rewrite) hasMappedBranch(node ast.Node) bool {
	for label := range r.labels {
		if hasLabelBranch(node, label) {
			return true
		}
	}
	return false
}

func (r *rewrite) header(statement ast.Stmt) []ast.Stmt {
	if !contains(statement, r.target) {
		return []ast.Stmt{statement}
	}
	return r.statementInner(statement)
}

func (r *rewrite) hasGoto(label *ast.LabeledStmt) bool {
	var node ast.Node = label
	for {
		parent := r.source.parents[node]
		if parent == nil {
			break
		}
		node = parent
		switch node.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			goto found
		}
	}
found:
	used := false
	ast.Inspect(node, func(n ast.Node) bool {
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Tok == token.GOTO && branch.Label.Name == label.Label.Name {
			used = true
		}
		return true
	})
	return used
}

func boolean(value bool) ast.Expr {
	op := token.NEQ
	if value {
		op = token.EQL
	}
	return &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: op, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
}
