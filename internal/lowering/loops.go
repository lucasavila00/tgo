package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) forStatement(s *ast.ForStmt) []ast.Stmt {
	copy := *s
	copy.Body = r.block(s.Body)
	condition, refs := r.expression(s.Cond)
	var post, init []ast.Stmt
	if s.Post != nil {
		post = r.header(s.Post)
	}
	if s.Init != nil {
		init = r.header(s.Init)
	}
	jumpingInit := r.initializerJumps(s)
	if !jumpingInit && len(condition) == 0 && len(post) <= 1 && len(init) <= 1 {
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
	var outer, body []ast.Stmt
	if jumpingInit {
		outer, copy.Init = r.loopBindings(s.Init, init)
	} else if len(init) != 0 {
		outer = append(outer, init[:len(init)-1]...)
		copy.Init = init[len(init)-1]
	}
	if jumpingInit || len(post) != 0 {
		first := r.name()
		outer = append(outer, variable(first, r.temporaryType(types.Typ[types.Bool], firstExpression(s)), boolean(true)))
		firstBody := []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.ASSIGN, Rhs: []ast.Expr{boolean(false)}}}
		if jumpingInit {
			firstBody = append(firstBody, init...)
		}
		body = append(body, &ast.IfStmt{Cond: first, Body: &ast.BlockStmt{List: firstBody}, Else: &ast.BlockStmt{List: post}})
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

// loopBindings keeps new variables in the Go loop header for per-iteration copies.
// The source initializer still runs once, inside the loop, before its condition.
func (r *rewrite) loopBindings(original ast.Stmt, init []ast.Stmt) ([]ast.Stmt, ast.Stmt) {
	assignment, ok := original.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE {
		return nil, nil
	}
	if r.bindings == nil {
		r.bindings = make(map[types.Object]string)
	}
	header := &ast.AssignStmt{Tok: token.DEFINE}
	var before []ast.Stmt
	for _, lhs := range assignment.Lhs {
		id := lhs.(*ast.Ident)
		if id.Name == "_" {
			continue
		}
		object := r.source.inventory.Checked.TypesInfo.Defs[id]
		bindingName := r.nameFor(id.Name)
		zeroValue := r.name()
		r.bindings[object] = bindingName.Name
		before = append(before, variable(zeroValue, r.temporaryType(object.Type(), id), nil))
		header.Lhs = append(header.Lhs, bindingName)
		header.Rhs = append(header.Rhs, zeroValue)
	}
	// The bindings now exist in the loop header; the source initializer stores them.
	init[len(init)-1].(*ast.AssignStmt).Tok = token.ASSIGN
	return before, header
}

func (r *rewrite) initializerJumps(loop *ast.ForStmt) bool {
	if loop.Init == nil {
		return false
	}
	found := false
	ast.Inspect(loop.Init, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
			return false
		}
		if branch, ok := node.(*ast.BranchStmt); ok && r.branchTarget(branch) == loop {
			found = true
		}
		return !found
	})
	return found
}

func (r *rewrite) realFunction(function *ast.FuncLit) bool {
	call, _ := r.source.inventory.Parents[function].(*ast.CallExpr)
	return r.conditionals[call] == nil && r.comprehensions[call] == nil
}

// branchTarget uses source parents and checked labels, before statements move.
func (r *rewrite) branchTarget(branch *ast.BranchStmt) ast.Stmt {
	if branch.Tok != token.BREAK && branch.Tok != token.CONTINUE {
		return nil
	}
	for node := r.source.inventory.Parents[branch]; node != nil; node = r.source.inventory.Parents[node] {
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
			return nil
		}
		if branch.Label != nil {
			if label, ok := node.(*ast.LabeledStmt); ok && r.source.inventory.Checked.TypesInfo.Defs[label.Label] == r.source.inventory.Checked.TypesInfo.Uses[branch.Label] {
				return label.Stmt
			}
			continue
		}
		switch node := node.(type) {
		case *ast.RangeStmt:
			if r.comprehensionRanges[node] && contains(node.X, branch) {
				continue
			}
			return node
		case *ast.ForStmt:
			return node
		case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			if branch.Tok == token.BREAK {
				return node.(ast.Stmt)
			}
		}
	}
	return nil
}
