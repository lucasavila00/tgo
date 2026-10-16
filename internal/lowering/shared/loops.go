package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// forStatement moves added header work into a loop while retaining initializer, post, and condition timing.
// It keeps continue targets and per-iteration bindings in the generated loop.
func (r *Context) forStatement(s *ast.ForStmt) []ast.Stmt {
	output := *s
	output.Body = r.Block(s.Body)
	conditionEvaluation := r.Expression(s.Cond)
	conditionStatements, conditionValues := conditionEvaluation.Statements, conditionEvaluation.Expressions()
	var postStatements []ast.Stmt
	if s.Post != nil {
		postStatements = r.Header(s.Post)
	}
	var initStatements []ast.Stmt
	if s.Init != nil {
		initStatements = r.Header(s.Init)
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
		outerStatements, loopInit = r.loopBindings(s.Init)
		output.Init = loopInit
		if loopInit != nil {
			last := len(initStatements) - 1
			assignment := *initStatements[last].(*ast.AssignStmt)
			assignment.Tok = token.ASSIGN
			initStatements = append([]ast.Stmt(nil), initStatements...)
			initStatements[last] = &assignment
		}
	} else if len(initStatements) != 0 {
		outerStatements = append(outerStatements, initStatements[:len(initStatements)-1]...)
		output.Init = initStatements[len(initStatements)-1]
	}
	var bodyStatements []ast.Stmt
	if jumpingInit || len(postStatements) != 0 {
		first := r.Name()
		outerStatements = append(
			outerStatements,
			Variable(first, r.TemporaryType(types.Typ[types.Bool], firstExpression(s)), Boolean(true)),
		)
		firstBody := []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{first}, Tok: token.ASSIGN, Rhs: []ast.Expr{Boolean(false)}}}
		if jumpingInit {
			firstBody = append(firstBody, initStatements...)
		}
		bodyStatements = append(
			bodyStatements,
			&ast.IfStmt{Cond: first, Body: &ast.BlockStmt{List: firstBody}, Else: &ast.BlockStmt{List: postStatements}},
		)
	}
	bodyStatements = append(bodyStatements, conditionStatements...)
	if len(conditionValues) != 0 {
		stop := &ast.IfStmt{
			Cond: &ast.UnaryExpr{Op: token.NOT, X: conditionValues[0]},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}},
		}
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
func (r *Context) loopBindings(original ast.Stmt) ([]ast.Stmt, ast.Stmt) {
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
		object := r.input.Inventory.Checked.TypesInfo.Defs[id]
		bindingName := r.NameFor(id.Name)
		zeroValue := r.Name()
		r.file.bindings[object] = bindingName.Name
		before = append(before, Variable(zeroValue, r.TemporaryType(object.Type(), id), nil))
		header.Lhs = append(header.Lhs, bindingName)
		header.Rhs = append(header.Rhs, zeroValue)
	}
	return before, header
}

// initializerJumps reports whether an initializer contains a source break or continue targeting this loop.
func (r *Context) initializerJumps(loop *ast.ForStmt) bool {
	if loop.Init == nil {
		return false
	}
	found := false
	ast.Inspect(loop.Init, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		if branch, ok := node.(*ast.BranchStmt); ok && r.branchTarget(branch) == loop {
			found = true
		}
		return !found
	})
	return found
}

// branchTarget reads the checked source target before statements move.
func (r *Context) branchTarget(branch *ast.BranchStmt) ast.Node {
	return r.input.Inventory.Checked.TypesInfo.BranchTargets[branch]
}
