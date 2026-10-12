package lowering

import (
	"go/ast"
	"go/token"
)

func (r *rewrite) block(body *ast.BlockStmt) *ast.BlockStmt {
	if body == nil || (!r.contains(body) && !r.hasMappedBranch(body)) {
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
	if branch, ok := statement.(*ast.BranchStmt); ok {
		if exit := r.breakExits[branch]; exit != nil {
			return []ast.Stmt{&ast.BranchStmt{Tok: token.GOTO, Label: exit}}
		}
	}
	if branch, ok := statement.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO {
		if name := r.labels[branch.Label.Name]; name != "" {
			copy := *branch
			copy.Label = ast.NewIdent(name)
			return []ast.Stmt{&copy}
		}
	}
	if !r.contains(statement) && !r.hasMappedBranch(statement) {
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
		return r.expressionStatement(s)
	case *ast.ReturnStmt:
		if record, found := r.shortReturns[s]; found {
			return r.shortReturnStatement(s, record)
		}
		copy := *s
		before, values := r.operands(s.Results)
		copy.Results = values
		return append(before, &copy)
	case *ast.AssignStmt:
		return r.assignmentStatement(s)
	case *ast.IncDecStmt:
		copy := *s
		before, place := r.place(s.X)
		copy.X = place
		return append(before, &copy)
	case *ast.DeclStmt:
		return r.declarationStatement(s)
	case *ast.IfStmt:
		return r.ifStatement(s)
	case *ast.ForStmt:
		return r.forStatement(s)
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
			call, _ := r.source.inventory.Parents[n].(*ast.CallExpr)
			if r.conditionals[call] == nil && r.comprehensions[call] == nil {
				return false
			}
		}
		if branch, ok := n.(*ast.BranchStmt); ok && branch.Label != nil && branch.Tok != token.GOTO && branch.Label.Name == label {
			found = true
		}
		return true
	})
	return found
}

func (r *rewrite) hasMappedBranch(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if branch, ok := n.(*ast.BranchStmt); ok && r.breakExits[branch] != nil {
			found = true
		}
		return !found
	})
	if found {
		return true
	}
	for label := range r.labels {
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
		parent := r.source.inventory.Parents[node]
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
