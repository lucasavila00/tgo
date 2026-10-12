package lowering

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

func (r *rewrite) expressionStatement(s *ast.ExprStmt) []ast.Stmt {
	statements, values := r.expression(s.X)
	if len(values) == 0 {
		if call, ok := s.X.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if object := r.input.source.inventory.Checked.TypesInfo.Uses[id]; object == types.Universe.Lookup("panic") {
					// Keep the terminating statement after unreachable inserted code.
					return append(statements, s)
				}
			}
		}
		return statements
	}
	output := *s
	output.X = values[0]
	if r.isTarget(s.X) {
		// A completed expression statement has no further operation.
		for _, value := range values {
			statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
		}
		return statements
	}
	return append(statements, &output)
}

func (r *rewrite) assignmentStatement(s *ast.AssignStmt) []ast.Stmt {
	output := *s
	var statements []ast.Stmt
	output.Lhs = make([]ast.Expr, len(s.Lhs))
	if s.Tok == token.DEFINE {
		output.Lhs = s.Lhs
	} else {
		for i, lhs := range s.Lhs {
			placeStatements, place := r.place(lhs)
			statements = append(statements, placeStatements...)
			output.Lhs[i] = place
		}
	}
	valueStatements, values := r.operands(s.Rhs)
	statements = append(statements, valueStatements...)
	output.Rhs = values
	deferred := false
	for _, place := range output.Lhs {
		deferred = deferred || r.deferredPlace(place)
	}
	if deferred {
		var savedValues []ast.Expr
		for _, value := range values {
			if tv := r.input.source.inventory.Checked.TypesInfo.Types[value]; tv.Value != nil || tv.IsNil() {
				savedValues = append(savedValues, value)
				continue
			}
			saveStatements, saved := r.save(value)
			statements = append(statements, saveStatements...)
			savedValues = append(savedValues, saved...)
		}
		return append(statements, r.orderedStores(output.Lhs, savedValues, output.Tok)...)
	}
	return append(statements, &output)
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
		valueSpec := *value
		statements, values := r.operands(value.Values)
		valueSpec.Values = values
		output = append(output, statements...)
		output = append(output, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: declaration.Tok, Specs: []ast.Spec{&valueSpec}}})
	}
	return output
}

func (r *rewrite) ifStatement(s *ast.IfStmt) []ast.Stmt {
	output := *s
	output.Body = r.block(s.Body)
	if s.Else != nil {
		statements := r.statement(s.Else)
		if len(statements) == 1 {
			output.Else = statements[0]
		} else {
			output.Else = &ast.BlockStmt{List: statements}
		}
	}
	var initStatements []ast.Stmt
	if s.Init != nil {
		initStatements = r.header(s.Init)
	}
	conditionStatements, conditionValues := r.expression(s.Cond)
	output.Cond = conditionValues[0]
	if len(conditionStatements) == 0 && len(initStatements) <= 1 {
		if len(initStatements) != 0 {
			output.Init = initStatements[0]
		}
		return []ast.Stmt{&output}
	}
	output.Init = nil
	statements := append(initStatements, conditionStatements...)
	return []ast.Stmt{&ast.BlockStmt{List: append(statements, &output)}}
}

func (r *rewrite) labeledStatement(s *ast.LabeledStmt) []ast.Stmt {
	output := *s
	jump := r.hasGoto(s)
	var inner *ast.Ident
	if jump && r.hasLabelBranch(s.Stmt, s.Label.Name) {
		inner = r.name()
		if r.functionState.labels == nil {
			r.functionState.labels = make(map[string]string)
		}
		r.functionState.labels[s.Label.Name] = inner.Name
		defer delete(r.functionState.labels, s.Label.Name)
	}
	statements := r.statement(s.Stmt)
	if target := r.functionState.controlTargets[s.Stmt]; target != nil {
		label := s.Label
		if jump {
			label = inner
		}
		if label != nil && labelUsed(statements, label.Name) {
			attachControlLabel(statements, target, label)
		}
		if jump {
			output.Stmt = &ast.BlockStmt{List: statements}
			return []ast.Stmt{&output}
		}
		return statements
	}
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
		output.Stmt = statements[0]
		return append([]ast.Stmt{&output}, statements[1:]...)
	case *ast.AssignStmt:
		if declaration.Tok == token.DEFINE {
			output.Stmt = statements[0]
			return append([]ast.Stmt{&output}, statements[1:]...)
		}
	}
	if !jump {
		if len(statements) == 1 {
			if block, ok := statements[0].(*ast.BlockStmt); ok {
				last := len(block.List) - 1
				output.Stmt = block.List[last]
				block.List[last] = &output
				return statements
			}
		}
		output.Stmt = statements[len(statements)-1]
		return append(statements[:len(statements)-1], &output)
	}
	if inner != nil {
		attachLabel(statements, inner)
	}
	output.Stmt = &ast.BlockStmt{List: statements}
	return []ast.Stmt{&output}
}
