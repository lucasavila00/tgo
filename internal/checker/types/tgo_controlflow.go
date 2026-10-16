package types

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
	"github.com/lucasavila00/tgo/internal/controlflow"
)

// branchFlow checks reachable paths through source statements, panic calls, and evaluated operands.
func (check *Checker) branchFlow(body *ast.BlockStmt) controlflow.Result {
	return controlflow.Branch(body, controlflow.Syntax{
		Terminates:  func(call *ast.CallExpr) bool { return check.isPanic[call] },
		Unevaluated: func(call *ast.CallExpr) bool { return check.Types[call].Value != nil },
	})
}

// expressionStatements visits native expressions in a statement's operands.
// Their statements stay in the current function; source function literals stop the walk.
func (check *Checker) expressionStatements(statement ast.Node, visit func(ast.Node)) {
	var first, second ast.Stmt
	switch statement := statement.(type) {
	case *ast.IfStmt:
		first = statement.Init
	case *ast.ForStmt:
		first, second = statement.Init, statement.Post
	case *ast.SwitchStmt:
		first = statement.Init
	case *ast.TypeSwitchStmt:
		first, second = statement.Init, statement.Assign
	case *ast.CommClause:
		first = statement.Comm
	}
	ast.Inspect(statement, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if child, ok := node.(ast.Stmt); ok && child != statement && child != first && child != second {
			return false
		}
		switch node.(type) {
		case *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
			visit(node)
			return false
		case *ast.FuncLit:
			return false
		}
		return true
	})
}

// conditionalFunctionExit checks exits that Go's statement grammar cannot see.
func (check *Checker) conditionalFunctionExit(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.IfExpr, *ast.SwitchExpr, *ast.ComprehensionExpr:
			found = true
			return false
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
	if !found || check.branchFlow(body).End {
		return false
	}
	if check.Info != nil {
		if check.TGoReturnBlocks == nil {
			check.TGoReturnBlocks = make(map[*ast.BlockStmt]bool)
		}
		check.TGoReturnBlocks[body] = true
	}
	return true
}

// rejectRangeCollectionContinue checks jumps before a range has its collection.
func (check *Checker) rejectRangeCollectionContinue(expr ast.Expr) {
	ast.Walk(rangeContinueVisitor{check: check}, expr)
}

type rangeContinueVisitor struct {
	check *Checker
	loops int
}

// Visit reports continue statements that cannot advance an enclosing range yet.
// It tracks nested loops and visits added-expression bodies, but stops at real function literals.
func (visitor rangeContinueVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil:
		return nil
	case *ast.FuncLit:
		return nil
	case *ast.ComprehensionExpr:
		for _, clause := range node.Clauses {
			switch clause := clause.(type) {
			case *ast.RangeClause:
				ast.Walk(visitor, clause.X)
				visitor.loops++
			case *ast.FilterClause:
				ast.Walk(visitor, clause.Cond)
			}
		}
		for _, output := range node.Output {
			ast.Walk(visitor, output)
		}
		return nil
	case *ast.ForStmt, *ast.RangeStmt:
		visitor.loops++
	case *ast.BranchStmt:
		if node.Tok == token.CONTINUE && node.Label == nil && visitor.loops == 0 {
			visitor.check.error(node, MisplacedContinue,
				"continue cannot advance a range before its collection is computed")
		}
	}
	return visitor
}

type valueSwitchBreakVisitor struct {
	check              *Checker
	nestedBreakTargets int
}

// Visit reports an unlabelled break that targets a value switch.
// It tracks nested break targets and stops at real function literals.
func (visitor valueSwitchBreakVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil, *ast.FuncLit:
		return nil
	case *ast.ComprehensionExpr:
		for _, clause := range node.Clauses {
			switch clause := clause.(type) {
			case *ast.RangeClause:
				ast.Walk(visitor, clause.X)
				visitor.nestedBreakTargets++
			case *ast.FilterClause:
				ast.Walk(visitor, clause.Cond)
			}
		}
		for _, output := range node.Output {
			ast.Walk(visitor, output)
		}
		return nil
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		visitor.nestedBreakTargets++
	case *ast.BranchStmt:
		if node.Tok == token.BREAK && node.Label == nil && visitor.nestedBreakTargets == 0 {
			visitor.check.error(node, InvalidSyntaxTree, "break cannot target a switch expression")
		}
	}
	return visitor
}

// pushControlTarget enters a source construct that accepts break or continue.
func (check *Checker) pushControlTarget(node ast.Node) {
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		check.controlTargets = append(check.controlTargets, node)
	}
}

// recordBranchTarget stores the current source target of an unlabelled jump.
func (check *Checker) recordBranchTarget(branch *ast.BranchStmt) {
	if branch.Label != nil || check.Info == nil {
		return
	}
	for i := len(check.controlTargets) - 1; i >= 0; i-- {
		target := check.controlTargets[i]
		switch branch.Tok {
		case token.BREAK:
			check.saveBranchTarget(branch, target)
			return
		case token.CONTINUE:
			switch target.(type) {
			case *ast.ForStmt, *ast.RangeStmt, *ast.RangeClause:
				check.saveBranchTarget(branch, target)
				return
			}
		}
	}
}

// saveBranchTarget records a checked source target without using byte positions.
func (check *Checker) saveBranchTarget(branch *ast.BranchStmt, target ast.Node) {
	if check.Info == nil {
		return
	}
	if check.BranchTargets == nil {
		check.BranchTargets = make(map[*ast.BranchStmt]ast.Node)
	}
	check.BranchTargets[branch] = target
}

// nativeHasBreak checks source statements and operands inside a native expression.
func (check *Checker) nativeHasBreak(node ast.Node, label string, implicit bool) bool {
	switch node := node.(type) {
	case *ast.IfExpr:
		if node.Init != nil && check.hasBreak(node.Init, label, implicit) ||
			check.expressionHasBreak(node.Cond, label, implicit) || check.hasBreak(node.Body, label, implicit) {
			return true
		}
		switch alternative := node.Else.(type) {
		case *ast.BlockStmt:
			return check.hasBreak(alternative, label, implicit)
		case *ast.IfExpr:
			return check.nativeHasBreak(alternative, label, implicit)
		}
	case *ast.SwitchExpr:
		return check.hasBreak(node.Switch, label, implicit)
	case *ast.ComprehensionExpr:
		return check.comprehensionHasBreak(node, label, implicit)
	}
	return false
}

// expressionHasBreak checks statements inside an expression's evaluated operands.
func (check *Checker) expressionHasBreak(expression ast.Expr, label string, implicit bool) bool {
	found := false
	check.expressionStatements(expression, func(node ast.Node) {
		found = found || check.nativeHasBreak(node, label, implicit)
	})
	return found
}

// comprehensionHasBreak keeps generator-source breaks with their enclosing targets.
func (check *Checker) comprehensionHasBreak(comprehension *ast.ComprehensionExpr, label string, implicit bool) bool {
	for _, clause := range comprehension.Clauses {
		switch clause := clause.(type) {
		case *ast.RangeClause:
			if check.expressionHasBreak(clause.X, label, implicit) {
				return true
			}
			implicit = false
		case *ast.FilterClause:
			if check.expressionHasBreak(clause.Cond, label, implicit) {
				return true
			}
		}
	}
	for _, output := range comprehension.Output {
		if check.expressionHasBreak(output, label, implicit) {
			return true
		}
	}
	return false
}
