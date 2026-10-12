package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) forStatement(s *ast.ForStmt) []ast.Stmt {
	output := *s
	output.Body = r.block(s.Body)
	conditionEvaluation := r.expression(s.Cond)
	conditionStatements, conditionValues := conditionEvaluation.statements, conditionEvaluation.expressions()
	var postStatements []ast.Stmt
	if s.Post != nil {
		postStatements = r.header(s.Post)
	}
	var initStatements []ast.Stmt
	if s.Init != nil {
		initStatements = r.header(s.Init)
	}
	jumpingInit := r.initializerJumps(s)
	if !jumpingInit && len(conditionStatements) == 0 && len(postStatements) <= 1 && len(initStatements) <= 1 {
		if len(conditionValues) != 0 {
			output.Cond = conditionValues[0]
		}
		if len(postStatements) != 0 {
			output.Post = postStatements[0]
		}
		if len(initStatements) != 0 {
			output.Init = initStatements[0]
		}
		return []ast.Stmt{&output}
	}
	var outerStatements []ast.Stmt
	if jumpingInit {
		var loopInit ast.Stmt
		outerStatements, loopInit = r.loopBindings(s.Init, initStatements)
		output.Init = loopInit
	} else if len(initStatements) != 0 {
		outerStatements = append(outerStatements, initStatements[:len(initStatements)-1]...)
		output.Init = initStatements[len(initStatements)-1]
	}
	var bodyStatements []ast.Stmt
	if jumpingInit || len(postStatements) != 0 {
		first := r.name()
		outerStatements = append(outerStatements, variable(first, r.temporaryType(types.Typ[types.Bool],
			firstExpression(s)), boolean(true)))
		firstBody := []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.ASSIGN,
			Rhs: []ast.Expr{boolean(false)}}}
		if jumpingInit {
			firstBody = append(firstBody, initStatements...)
		}
		bodyStatements = append(bodyStatements, &ast.IfStmt{Cond: first, Body: &ast.BlockStmt{List: firstBody},
			Else: &ast.BlockStmt{List: postStatements}})
	}
	bodyStatements = append(bodyStatements, conditionStatements...)
	if len(conditionValues) != 0 {
		stop := &ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: conditionValues[0]},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}}}
		bodyStatements = append(bodyStatements, stop)
	}
	bodyStatements = append(bodyStatements, output.Body)
	output.Cond = nil
	output.Post = nil
	output.Body = &ast.BlockStmt{List: bodyStatements}
	return []ast.Stmt{&ast.BlockStmt{List: append(outerStatements, &output)}}
}

// loopBindings keeps new variables in the Go loop header for per-iteration copies.
// The source initializer still runs once, inside the loop, before its condition.
func (r *rewrite) loopBindings(original ast.Stmt, init []ast.Stmt) ([]ast.Stmt, ast.Stmt) {
	assignment, ok := original.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE {
		return nil, nil
	}
	if r.file.bindings == nil {
		r.file.bindings = make(map[types.Object]string)
	}
	header := &ast.AssignStmt{Tok: token.DEFINE}
	var before []ast.Stmt
	for _, lhs := range assignment.Lhs {
		id := lhs.(*ast.Ident)
		if id.Name == "_" {
			continue
		}
		object := r.input.source.inventory.Checked.TypesInfo.Defs[id]
		bindingName := r.nameFor(id.Name)
		zeroValue := r.name()
		r.file.bindings[object] = bindingName.Name
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
	call, _ := r.input.source.inventory.Parents[function].(*ast.CallExpr)
	return r.input.conditionals[call] == nil && r.input.comprehensions[call] == nil
}

// branchTarget uses source parents and checked labels, before statements move.
func (r *rewrite) branchTarget(branch *ast.BranchStmt) ast.Stmt {
	if branch.Tok != token.BREAK && branch.Tok != token.CONTINUE {
		return nil
	}
	for node := r.input.source.inventory.Parents[branch]; node != nil; node = r.input.source.inventory.Parents[node] {
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
			return nil
		}
		if branch.Label != nil {
			info := r.input.source.inventory.Checked.TypesInfo
			if label, ok := node.(*ast.LabeledStmt); ok &&
				info.Defs[label.Label] == info.Uses[branch.Label] {
				return label.Stmt
			}
			continue
		}
		switch node := node.(type) {
		case *ast.RangeStmt:
			if r.functionState.comprehensionRanges[node] && contains(node.X, branch) {
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
