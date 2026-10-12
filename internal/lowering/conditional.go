package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

func (r *rewrite) valueExpression(call *ast.CallExpr, statement ast.Stmt) expressionResult {
	resultVariable := r.name()
	resultType := types.Default(r.typeOf(call))
	declaration := r.resultStorage(resultVariable, resultType, call, nil)
	branchStatements := r.valueStatement(statement, resultVariable)
	r.completeExpression(call)
	return r.result(append([]ast.Stmt{declaration}, branchStatements...), []ast.Expr{resultVariable})
}

func (r *rewrite) conditionalStatement(statement *ast.IfStmt, result *ast.Ident) []ast.Stmt {
	output := *statement
	output.Body = r.conditionalBlock(statement.Body, result)
	switch alternative := statement.Else.(type) {
	case *ast.BlockStmt:
		output.Else = r.conditionalBlock(alternative, result)
	case *ast.IfStmt:
		alternativeStatements := r.conditionalStatement(alternative, result)
		if len(alternativeStatements) == 1 {
			output.Else = alternativeStatements[0]
		} else {
			output.Else = &ast.BlockStmt{List: alternativeStatements}
		}
	}
	var initStatements []ast.Stmt
	if statement.Init != nil {
		initStatements = r.header(statement.Init)
	}
	conditionEvaluation := r.expression(statement.Cond)
	conditionStatements, conditionValues := conditionEvaluation.statements, conditionEvaluation.expressions()
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

func (r *rewrite) conditionalBlock(body *ast.BlockStmt, result *ast.Ident) *ast.BlockStmt {
	output := *body
	output.List = nil
	if completes, recorded := r.input.source.inventory.Checked.TypesInfo.TGoBranches[body]; recorded && !completes {
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
			output.List = append(output.List, r.conditionalTail(statement, result)...)
		} else {
			output.List = append(output.List, r.statement(statement)...)
		}
	}
	return &output
}

func (r *rewrite) conditionalTail(statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.LabeledStmt:
		return r.conditionalLabeledTail(statement, result)
	case *ast.IfStmt:
		return r.conditionalStatement(statement, result)
	case *ast.SwitchStmt:
		return r.valueSwitchStatement(statement, result)
	case *ast.TypeSwitchStmt:
		return r.valueTypeSwitchStatement(statement, result)
	case *ast.ExprStmt:
		return r.conditionalExpressionTail(statement, result)
	}
	return r.statement(statement)
}

func (r *rewrite) valueStatement(statement ast.Stmt, result *ast.Ident) []ast.Stmt {
	switch statement := statement.(type) {
	case *ast.IfStmt:
		return r.conditionalStatement(statement, result)
	case *ast.SwitchStmt:
		return r.valueSwitchStatement(statement, result)
	case *ast.TypeSwitchStmt:
		return r.valueTypeSwitchStatement(statement, result)
	}
	return nil
}

func (r *rewrite) conditionalLabeledTail(statement *ast.LabeledStmt, result *ast.Ident) []ast.Stmt {
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
		return r.statement(statement)
	}
	labeled := *statement
	statements := r.conditionalTail(statement.Stmt, result)
	if len(statements) == 1 {
		labeled.Stmt = statements[0]
	} else {
		labeled.Stmt = &ast.BlockStmt{List: statements}
	}
	return []ast.Stmt{&labeled}
}

func (r *rewrite) conditionalExpressionTail(statement *ast.ExprStmt, result *ast.Ident) []ast.Stmt {
	if tv := r.input.source.inventory.Checked.TypesInfo.Types[statement.X]; !tv.IsVoid() {
		evaluation := r.expression(statement.X)
		statements, values := evaluation.statements, evaluation.expressions()
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
	return r.statement(statement)
}
