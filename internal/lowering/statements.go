package lowering

import (
	"go/ast"
	"go/token"
)

func (r *rewrite) block(body *ast.BlockStmt) *ast.BlockStmt {
	if body == nil || (!r.contains(body) && !r.hasMappedBranch(body)) {
		return body
	}
	output := *body
	output.List = nil
	for _, statement := range body.List {
		output.List = append(output.List, r.statement(statement)...)
	}
	return &output
}

func (r *rewrite) statement(statement ast.Stmt) []ast.Stmt {
	if branch, ok := statement.(*ast.BranchStmt); ok {
		if exit := r.functionState.breakExits[branch]; exit != nil {
			return []ast.Stmt{&ast.BranchStmt{Tok: token.GOTO, Label: exit}}
		}
	}
	if branch, ok := statement.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO {
		if name := r.functionState.labels[branch.Label.Name]; name != "" {
			output := *branch
			output.Label = ast.NewIdent(name)
			return []ast.Stmt{&output}
		}
	}
	if !r.contains(statement) && !r.hasMappedBranch(statement) {
		return []ast.Stmt{statement}
	}
	output := r.statementInner(statement)
	if len(output) <= 1 || !r.functionState.hasGoto {
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
		output := *s
		statementsEvaluation := r.operands([]ast.Expr{s.Chan, s.Value})
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		output.Chan = values[0]
		output.Value = values[1]
		return append(statements, &output)
	case *ast.ExprStmt:
		return r.expressionStatement(s)
	case *ast.ReturnStmt:
		if record, found := r.input.shortReturns[s]; found {
			return r.shortReturnStatement(s, record)
		}
		output := *s
		statementsEvaluation := r.operands(s.Results)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		output.Results = values
		return append(statements, &output)
	case *ast.AssignStmt:
		return r.assignmentStatement(s)
	case *ast.IncDecStmt:
		output := *s
		statements, place := r.place(s.X)
		output.X = place
		return append(statements, &output)
	case *ast.DeclStmt:
		return r.declarationStatement(s)
	case *ast.IfStmt:
		return r.ifStatement(s)
	case *ast.ForStmt:
		return r.forStatement(s)
	case *ast.GoStmt:
		output := *s
		statementsEvaluation := r.expression(s.Call)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		output.Call = values[0].(*ast.CallExpr)
		return append(statements, &output)
	case *ast.DeferStmt:
		output := *s
		statementsEvaluation := r.expression(s.Call)
		statements, values := statementsEvaluation.Statements, statementsEvaluation.Expressions()
		output.Call = values[0].(*ast.CallExpr)
		return append(statements, &output)
	case *ast.LabeledStmt:
		return r.labeledStatement(s)
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

func (r *rewrite) hasLabelBranch(node ast.Node, label string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			call, _ := r.input.source.inventory.Parents[n].(*ast.CallExpr)
			if r.input.conditionals[call] == nil && r.input.comprehensions[call] == nil {
				return false
			}
		}
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO &&
			branch.Label.Name == label {
			found = true
		}
		return true
	})
	return found
}

func (r *rewrite) hasMappedBranch(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if branch, ok := n.(*ast.BranchStmt); ok && r.functionState.breakExits[branch] != nil {
			found = true
		}
		return !found
	})
	if found {
		return true
	}
	for label := range r.functionState.labels {
		if r.hasLabelBranch(node, label) {
			return true
		}
	}
	return false
}

func (r *rewrite) header(statement ast.Stmt) []ast.Stmt {
	if !r.contains(statement) && !r.hasMappedBranch(statement) {
		return []ast.Stmt{statement}
	}
	return r.statementInner(statement)
}

func (r *rewrite) hasGoto(label *ast.LabeledStmt) bool {
	var node ast.Node = label
	for {
		parent := r.input.source.inventory.Parents[node]
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
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Tok == token.GOTO &&
			branch.Label.Name == label.Label.Name {
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
	return &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.INT, Value: "0"}, Op: op,
		Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
}

// functionHasGoto excludes nested functions, which have their own jump scope.
func (r *rewrite) functionHasGoto(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncLit); ok && r.realFunction(function) {
			return false
		}
		if branch, ok := node.(*ast.BranchStmt); ok && branch.Tok == token.GOTO {
			found = true
		}
		return !found
	})
	return found
}
