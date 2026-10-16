package valueexpr

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Expression produces a checked value from the reached branch.
func Expression(r *shared.Context, expression ast.Expr) shared.Result {
	resultVariable := r.Name()
	resultType := types.Default(r.TypeOf(expression))
	declaration := r.ResultStorage(resultVariable, resultType, expression, nil)
	var branchStatements []ast.Stmt
	switch expression := expression.(type) {
	case *ast.IfExpr:
		branchStatements = conditionalExpression(r, expression, resultVariable)
	case *ast.SwitchExpr:
		branchStatements = valueStatement(r, expression.Switch, resultVariable)
	}
	r.CompleteExpression(expression)
	return r.ResultOf(append([]ast.Stmt{declaration}, branchStatements...), []ast.Expr{resultVariable})
}

// conditionalExpression builds Go branches from a value-producing if.
func conditionalExpression(r *shared.Context, expression *ast.IfExpr, result *ast.Ident) []ast.Stmt {
	body := conditionalBlock(r, expression.Body, result)
	var alternative ast.Stmt
	switch branch := expression.Else.(type) {
	case *ast.BlockStmt:
		alternative = conditionalBlock(r, branch, result)
	case *ast.IfExpr:
		statements := conditionalExpression(r, branch, result)
		r.CompleteExpression(branch)
		if len(statements) == 1 {
			alternative = statements[0]
		} else {
			alternative = &ast.BlockStmt{List: statements}
		}
	}
	source := &ast.IfStmt{If: expression.If, Init: expression.Init, Cond: expression.Cond}
	return r.IfStatement(source, body, alternative)
}

// conditionalStatement lowers if branches so each reached value tail writes the shared result local.
func conditionalStatement(r *shared.Context, statement *ast.IfStmt, result *ast.Ident) []ast.Stmt {
	body := conditionalBlock(r, statement.Body, result)
	var alternative ast.Stmt
	switch sourceAlternative := statement.Else.(type) {
	case *ast.BlockStmt:
		alternative = conditionalBlock(r, sourceAlternative, result)
	case *ast.IfStmt:
		statements := conditionalStatement(r, sourceAlternative, result)
		if len(statements) == 1 {
			alternative = statements[0]
		} else {
			alternative = &ast.BlockStmt{List: statements}
		}
	}
	return r.IfStatement(statement, body, alternative)
}

// conditionalBlock lowers branch statements and uses the last nonempty statement to supply the result.
// It omits result storage for branches that cannot complete normally.
func conditionalBlock(r *shared.Context, body *ast.BlockStmt, result *ast.Ident) *ast.BlockStmt {
	output := *body
	output.List = nil
	if completes, recorded := r.Inventory().Checked.TypesInfo.TGoBranches[body]; recorded && !completes {
		result = nil
	}
	last := len(body.List) - 1
	for last >= 0 {
		if _, empty := body.List[last].(*ast.EmptyStmt); !empty {
			break
		}
		last--
	}
	for i, statement := range body.List {
		if i == last {
			output.List = append(output.List, conditionalTail(r, statement, result)...)
		} else {
			output.List = append(output.List, r.Statement(statement)...)
		}
	}
	return &output
}

// conditionalTail lowers a branch tail as a value assignment or a nested value-producing control statement.
func conditionalTail(r *shared.Context, statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.LabeledStmt:
		return conditionalLabeledTail(r, statement, result)
	case *ast.IfStmt:
		return conditionalStatement(r, statement, result)
	case *ast.SwitchStmt:
		return r.ValueSwitchStatement(statement, result, conditionalBlock)
	case *ast.TypeSwitchStmt:
		return r.ValueTypeSwitchStatement(statement, result, conditionalBlock)
	case *ast.ExprStmt:
		return conditionalExpressionTail(r, statement, result)
	}
	return r.Statement(statement)
}

// valueStatement selects if, switch, or type-switch lowering with the supplied result destination.
func valueStatement(r *shared.Context, statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.IfStmt:
		return conditionalStatement(r, statement, result)
	case *ast.SwitchStmt:
		return r.ValueSwitchStatement(statement, result, conditionalBlock)
	case *ast.TypeSwitchStmt:
		return r.ValueTypeSwitchStatement(statement, result, conditionalBlock)
	}
	return nil
}

// conditionalLabeledTail keeps source labels attached when a value-producing tail expands into multiple statements.
func conditionalLabeledTail(r *shared.Context, statement *ast.LabeledStmt, result *ast.Ident) []ast.Stmt {
	tail := statement.Stmt
	for {
		label, ok := tail.(*ast.LabeledStmt)
		if !ok {
			break
		}
		tail = label.Stmt
	}
	switch tail.(type) {
	case *ast.ExprStmt, *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
	default:
		return r.Statement(statement)
	}
	labeled := *statement
	statements := conditionalTail(r, statement.Stmt, result)
	if len(statements) == 1 {
		labeled.Stmt = statements[0]
	} else {
		labeled.Stmt = &ast.BlockStmt{List: statements}
	}
	return []ast.Stmt{&labeled}
}

// conditionalExpressionTail lowers the tail expression and stores or discards its value as the branch requires.
func conditionalExpressionTail(r *shared.Context, statement *ast.ExprStmt, result *ast.Ident) []ast.Stmt {
	if tv := r.Inventory().Checked.TypesInfo.Types[statement.X]; !tv.IsVoid() {
		evaluation := r.Expression(statement.X)
		statements, values := evaluation.Statements, evaluation.Expressions()
		if result == nil {
			for _, value := range values {
				if tv.Value != nil {
					value = &ast.BinaryExpr{X: value, Op: token.EQL, Y: value}
				}
				if tv.IsNil() {
					value = &ast.CallExpr{Fun: &ast.InterfaceType{Methods: &ast.FieldList{}}, Args: []ast.Expr{value}}
				}
				statements = append(
					statements,
					&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}},
				)
			}
			return statements
		}
		return append(statements, &ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.ASSIGN, Rhs: values})
	}
	return r.Statement(statement)
}
