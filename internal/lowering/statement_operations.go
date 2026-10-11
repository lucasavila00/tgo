package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) expressionStatement(s *ast.ExprStmt) []ast.Stmt {
	before, refs := r.expression(s.X)
	if len(refs) == 0 {
		if call, ok := s.X.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if object := r.source.checked.TypesInfo.Uses[id]; object == types.Universe.Lookup("panic") {
					// Keep the terminating statement after unreachable inserted code.
					return append(before, s)
				}
			}
		}
		return before
	}
	copy := *s
	copy.X = refs[0]
	if r.isTarget(s.X) {
		// A completed expression statement has no further operation.
		for _, ref := range refs {
			before = append(before, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ref}})
		}
		return before
	}
	return append(before, &copy)
}

func (r *rewrite) assignmentStatement(s *ast.AssignStmt) []ast.Stmt {
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
	deferred := false
	for _, place := range copy.Lhs {
		deferred = deferred || r.deferredPlace(place)
	}
	if deferred {
		var refs []ast.Expr
		for _, value := range values {
			if tv := r.source.checked.TypesInfo.Types[value]; tv.Value != nil || tv.IsNil() {
				refs = append(refs, value)
				continue
			}
			setup, saved := r.save(value)
			before = append(before, setup...)
			refs = append(refs, saved...)
		}
		return append(before, r.orderedStores(copy.Lhs, refs, copy.Tok)...)
	}
	return append(before, &copy)
}

func (r *rewrite) declarationStatement(s *ast.DeclStmt) []ast.Stmt {
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
}

func (r *rewrite) ifStatement(s *ast.IfStmt) []ast.Stmt {
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
}

func (r *rewrite) forStatement(s *ast.ForStmt) []ast.Stmt {
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
		outer = append(outer, variable(first, r.temporaryType(types.Typ[types.Bool], firstExpression(s)), boolean(true)))
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
}

func (r *rewrite) labeledStatement(s *ast.LabeledStmt) []ast.Stmt {
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
	// Keep source declarations in their original scope, including nested labels.
	declaration := s.Stmt
	for {
		label, ok := declaration.(*ast.LabeledStmt)
		if !ok {
			break
		}
		declaration = label.Stmt
	}
	switch declaration := declaration.(type) {
	case *ast.DeclStmt:
		copy.Stmt = statements[0]
		return append([]ast.Stmt{&copy}, statements[1:]...)
	case *ast.AssignStmt:
		if declaration.Tok == token.DEFINE {
			copy.Stmt = statements[0]
			return append([]ast.Stmt{&copy}, statements[1:]...)
		}
	}
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
