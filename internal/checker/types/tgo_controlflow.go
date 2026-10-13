package types

import (
	"go/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
	"github.com/lucasavila00/tgo/internal/controlflow"
)

func (check *Checker) branchFlow(body *ast.BlockStmt) controlflow.Result {
	return controlflow.Branch(body, controlflow.Syntax{
		RangeSourceOutside: func(loop *ast.RangeStmt) bool { return check.tgo.comprehensionRanges[loop] },
		Statements:         check.operationStatements,
		Terminates:         func(call *ast.CallExpr) bool { return check.isPanic[call] },
		Unevaluated:        func(call *ast.CallExpr) bool { return check.Types[call].Value != nil },
	})
}

// expressionStatements visits source statements inside added expressions. Nested statements
// are visited by the caller; real function literals belong to another function.
func (check *Checker) expressionStatements(statement ast.Stmt, visit func(ast.Stmt)) {
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
		if call, ok := node.(*ast.CallExpr); ok {
			if body := check.operationStatements(call); body != nil {
				visit(body)
				return false
			}
		}
		_, function := node.(*ast.FuncLit)
		return !function
	})
}

// conditionalFunctionExit checks exits that Go's statement grammar cannot see.
func (check *Checker) conditionalFunctionExit(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if operation, exists := check.tgoOperation(call); exists && (operation.Conditional ||
				operation.Comprehension) {
				found = true
				return false
			}
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

func (visitor rangeContinueVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil:
		return nil
	case *ast.FuncLit:
		return nil
	case *ast.CallExpr:
		if body := visitor.check.operationStatements(node); body != nil {
			ast.Walk(visitor, body)
			return nil
		}
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

func (visitor valueSwitchBreakVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil, *ast.FuncLit:
		return nil
	case *ast.CallExpr:
		if body := visitor.check.operationStatements(node); body != nil {
			ast.Walk(visitor, body)
			return nil
		}
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		visitor.nestedBreakTargets++
	case *ast.BranchStmt:
		if node.Tok == token.BREAK && node.Label == nil && visitor.nestedBreakTargets == 0 {
			visitor.check.error(node, InvalidSyntaxTree, "break cannot target a switch expression")
		}
	}
	return visitor
}
