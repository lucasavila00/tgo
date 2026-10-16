package lowering

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/lowering/comprehensions"
	"github.com/lucasavila00/tgo/internal/lowering/propagation"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"github.com/lucasavila00/tgo/internal/lowering/valueexpr"
	"go/token"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

// newRewrite connects the checked source to the rule dispatchers and creates fresh rewrite state.
func newRewrite(source *Source, input shared.Input) *shared.Context {
	input.Inventory = &source.inventory
	return shared.New(input, shared.Dispatch{Expression: expression, Statement: statement})
}

// statement selects propagation without success storage for discarded results; it lowers other statements normally.
func statement(r *shared.Context, stmt ast.Stmt) []ast.Stmt {
	if expr, ok := stmt.(*ast.ExprStmt); ok {
		if statements, handled := discardedPropagation(r, expr.X); handled {
			return statements
		}
	}
	if assignment, ok := stmt.(*ast.AssignStmt); ok && assignment.Tok == token.ASSIGN &&
		len(assignment.Rhs) == 1 && shared.BlankDestinations(assignment.Lhs) {
		if statements, handled := discardedPropagation(r, assignment.Rhs[0]); handled {
			return statements
		}
	}
	return r.StatementSyntax(stmt)
}

// expression lowers selected expressions and retains checked values through rule dispatch and parentheses.
func expression(r *shared.Context, expr ast.Expr) shared.Result {
	if expr == nil {
		return r.ResultOf(nil, nil)
	}
	if !r.Contains(expr) {
		return r.ResultOf(nil, []ast.Expr{expr})
	}
	if result, handled := r.EnumUnevaluated(expr); handled {
		return result
	}
	if result, special := specialExpression(r, expr); special {
		return result
	}
	if parenthesized, ok := expr.(*ast.ParenExpr); ok {
		inner := r.Expression(parenthesized.X)
		if len(inner.Values) != 1 {
			return inner
		}
		output := *parenthesized
		output.X = inner.Values[0].Expression
		return r.FinishExpression(expr, inner.Statements, &output)
	}
	statements, result := r.ExpressionSyntax(expr)
	return r.FinishExpression(expr, statements, result)
}

// specialExpression selects the propagation, comprehension, or value-expression rule from the checked source records.
func specialExpression(r *shared.Context, expr ast.Expr) (shared.Result, bool) {
	switch expression := expr.(type) {
	case *ast.PropagateExpr:
		return propagation.Expression(r, expression, false), true
	case *ast.ComprehensionExpr:
		return comprehensions.Expression(r, expression), true
	case *ast.IfExpr, *ast.SwitchExpr:
		return valueexpr.Expression(r, expression), true
	}
	return shared.Result{}, false
}

// discardedPropagation keeps the call and its error check without success storage.
func discardedPropagation(r *shared.Context, expression ast.Expr) ([]ast.Stmt, bool) {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	if operation, ok := expression.(*ast.PropagateExpr); ok {
		evaluation := propagation.Expression(r, operation, true)
		return evaluation.Statements, true
	}
	return nil, false
}
