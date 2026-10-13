package valueexpr

import (
	"github.com/lucasavila00/tgo/internal/lowering/shared"
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// Expression produces a checked value from the reached branch.
func Expression(r *shared.Context, call *ast.CallExpr, statement ast.Stmt) shared.Result {
	resultVariable := r.Name()
	resultType := types.Default(r.TypeOf(call))
	declaration := r.ResultStorage(resultVariable, resultType, call, nil)
	branchStatements := valueStatement(r, statement, resultVariable)
	r.CompleteExpression(call)
	return r.ResultOf(append([]ast.Stmt{declaration}, branchStatements...), []ast.Expr{resultVariable})
}

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
