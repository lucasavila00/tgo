package shared

import (
	"github.com/lucasavila00/tgo/internal/checker/types"
	"go/ast"
	"go/token"
)

// expressionStatement lowers an expression used for its effects and discards any stored results.
func (r *Context) expressionStatement(s *ast.ExprStmt) []ast.Stmt {
	statementsEvaluation := r.Expression(s.X)
	statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	if len(values) == 0 {
		if call, ok := s.X.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if object := r.input.Inventory.Checked.TypesInfo.Uses[id]; object == types.Universe.Lookup("panic") {
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
			statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")},
				Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
		}
		return statements
	}
	return append(statements, &output)
}

// BlankDestinations reports whether all assignment destinations discard values.
func BlankDestinations(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		identifier, ok := expression.(*ast.Ident)
		if !ok || identifier.Name != "_" {
			return false
		}
	}
	return len(expressions) != 0
}

// assignmentStatement prepares destinations and values in source order before emitting assignment stores.
func (r *Context) assignmentStatement(s *ast.AssignStmt) []ast.Stmt {
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
	valueEvaluation := r.Operands(s.Rhs)
	valueStatements, values := valueEvaluation.Statements, valueEvaluation.Expressions()
	statements = append(statements, valueStatements...)
	output.Rhs = values
	deferred := false
	for _, place := range output.Lhs {
		deferred = deferred || r.deferredPlace(place)
	}
	if deferred {
		var savedValues []ast.Expr
		for _, value := range values {
			if tv := r.input.Inventory.Checked.TypesInfo.Types[value]; tv.Value != nil || tv.IsNil() {
				savedValues = append(savedValues, value)
				continue
			}
			saveEvaluation := r.Save(r.ValueOf(value))
			saveStatements, saved := saveEvaluation.Statements, saveEvaluation.Expressions()
			statements = append(statements, saveStatements...)
			savedValues = append(savedValues, saved...)
		}
		return append(statements, r.orderedStores(output.Lhs, savedValues, output.Tok)...)
	}
	return append(statements, &output)
}

// declarationStatement lowers declaration initializers in order and keeps their statements before each declaration.
func (r *Context) declarationStatement(s *ast.DeclStmt) []ast.Stmt {
	declaration := s.Decl.(*ast.GenDecl)
	var output []ast.Stmt
	for _, spec := range declaration.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			output = append(output, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: declaration.Tok, Specs: []ast.Spec{spec}}})
			continue
		}
		valueSpec := *value
		statementsEvaluation := r.Operands(value.Values)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		valueSpec.Values = values
		output = append(output, statements...)
		output = append(output, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: declaration.Tok,
			Specs: []ast.Spec{&valueSpec}}})
	}
	return output
}

// ifStatement lowers both branch bodies and combines them with the rewritten initializer and condition.
func (r *Context) ifStatement(statement *ast.IfStmt) []ast.Stmt {
	body := r.Block(statement.Body)
	alternative := statement.Else
	if alternative != nil {
		statements := r.Statement(alternative)
		if len(statements) == 1 {
			alternative = statements[0]
		} else {
			alternative = &ast.BlockStmt{List: statements}
		}
	}
	return r.IfStatement(statement, body, alternative)
}

// IfStatement combines rewritten branches with the source initializer and condition.
func (r *Context) IfStatement(statement *ast.IfStmt, body *ast.BlockStmt, alternative ast.Stmt) []ast.Stmt {
	output := *statement
	output.Body = body
	output.Else = alternative
	var initStatements []ast.Stmt
	if statement.Init != nil {
		initStatements = r.Header(statement.Init)
	}
	conditionEvaluation := r.Expression(statement.Cond)
	conditionStatements, conditionValues := conditionEvaluation.Statements, conditionEvaluation.Expressions()
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

// labeledStatement keeps goto entry and labeled break or continue targets valid when one statement expands.
func (r *Context) labeledStatement(s *ast.LabeledStmt) []ast.Stmt {
	output := *s
	jump := r.hasGoto(s)
	var inner *ast.Ident
	if jump && r.hasLabelBranch(s.Stmt, s.Label.Name) {
		inner = r.Name()
		if r.functionState.labels == nil {
			r.functionState.labels = make(map[string]string)
		}
		r.functionState.labels[s.Label.Name] = inner.Name
		defer delete(r.functionState.labels, s.Label.Name)
	}
	statements := r.Statement(s.Stmt)
	if target := r.functionState.controlTargets[s.Stmt]; target != nil {
		label := s.Label
		if jump {
			label = inner
		}
		if label != nil && labelUsed(statements, label.Name) {
			attachControlLabel(statements, target, label)
		}
		if jump {
			output.Stmt = scopedBlock(statements)
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
		return labelFinalStatement(&output, statements)
	}
	if inner != nil {
		attachLabel(statements, inner)
	}
	output.Stmt = scopedBlock(statements)
	return []ast.Stmt{&output}
}

// scopedBlock reuses a sole block or puts the statement list in a new block.
func scopedBlock(statements []ast.Stmt) *ast.BlockStmt {
	if len(statements) == 1 {
		if block, ok := statements[0].(*ast.BlockStmt); ok {
			return block
		}
	}
	return &ast.BlockStmt{List: statements}
}

// labelFinalStatement attaches a control label to the last generated statement, leaving setup before it.
func labelFinalStatement(output *ast.LabeledStmt, statements []ast.Stmt) []ast.Stmt {
	if len(statements) == 1 {
		if block, ok := statements[0].(*ast.BlockStmt); ok {
			last := len(block.List) - 1
			output.Stmt = block.List[last]
			block.List[last] = output
			return statements
		}
	}
	output.Stmt = statements[len(statements)-1]
	return append(statements[:len(statements)-1], output)
}
