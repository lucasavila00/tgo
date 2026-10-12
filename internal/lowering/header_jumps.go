package lowering

import (
	"go/ast"
	"go/token"
)

// prepareBreakExit keeps source break targets when operand code moves before a statement.
func (r *rewrite) prepareBreakExit(target ast.Stmt, roots ...ast.Node) *ast.Ident {
	var exit *ast.Ident
	for _, root := range roots {
		if root == nil {
			continue
		}
		ast.Inspect(root, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncLit); ok && r.realFunction(fn) {
				return false
			}
			branch, ok := node.(*ast.BranchStmt)
			if ok && branch.Tok == token.BREAK && r.branchTarget(branch) == target {
				if exit == nil {
					exit = r.name()
					if r.functionState.breakExits == nil {
						r.functionState.breakExits = make(map[*ast.BranchStmt]*ast.Ident)
					}
				}
				r.functionState.breakExits[branch] = exit
			}
			return true
		})
	}
	return exit
}

// withBreakExit puts the exit outside all operand and statement locals.
func (r *rewrite) withBreakExit(source, target ast.Stmt, output []ast.Stmt, exit *ast.Ident) []ast.Stmt {
	if exit == nil {
		return output
	}
	if r.functionState.controlTargets == nil {
		r.functionState.controlTargets = make(map[ast.Stmt]ast.Stmt)
	}
	r.functionState.controlTargets[source] = target
	inner := &ast.BlockStmt{List: output}
	if len(output) == 1 {
		if block, ok := output[0].(*ast.BlockStmt); ok {
			inner = block
		}
	}
	return []ast.Stmt{&ast.BlockStmt{List: []ast.Stmt{inner, &ast.LabeledStmt{Label: exit, Stmt: &ast.EmptyStmt{Implicit: true}}}}}
}

func labelUsed(statements []ast.Stmt, label string) bool {
	found := false
	for _, statement := range statements {
		ast.Inspect(statement, func(node ast.Node) bool {
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}
			if branch, ok := node.(*ast.BranchStmt); ok && branch.Label != nil && branch.Label.Name == label {
				found = true
			}
			return true
		})
	}
	return found
}

func attachControlLabel(statements []ast.Stmt, target ast.Stmt, label *ast.Ident) {
	for i, statement := range statements {
		if statement == target {
			statements[i] = &ast.LabeledStmt{Label: label, Stmt: target}
			return
		}
		if block, ok := statement.(*ast.BlockStmt); ok {
			attachControlLabel(block.List, target, label)
		}
	}
}
