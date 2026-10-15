package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
)

// shortReturnStatement supplies omitted results without a function wrapper.
func (r *Context) shortReturnStatement(statement *ast.ShortReturnStmt) []ast.Stmt {
	statementsEvaluation := r.Operands(statement.Results)
	statements, returnValues := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	if statement.ZeroSuccess {
		zeroValues := r.PrepareFailureResults()
		returnValues = append(zeroValues, returnValues...)
	} else {
		_, tuple := statementsEvaluation.Values[0].Type.(*types.Tuple)
		if len(returnValues) == 1 && tuple {
			tupleStatements, tupleResults := r.shortReturnTuple(returnValues[0], statement.Results[0])
			statements = append(statements, tupleStatements...)
			returnValues = tupleResults
		}
		nilStatements, nilResult := r.shortReturnNil(statement)
		statements = append(statements, nilStatements...)
		returnValues = append(returnValues, nilResult)
	}
	returnStatement := ast.ReturnStmt{Return: statement.Return, Results: returnValues}
	r.request.loweredReturns[statement] = true
	return append(statements, &returnStatement)
}

// shortReturnNil supplies nil for the omitted error result, using typed zero storage when source hides nil.
func (r *Context) shortReturnNil(statement *ast.ShortReturnStmt) ([]ast.Stmt, ast.Expr) {
	scope := r.input.Inventory.Checked.TypesInfo.Scopes[statement]
	nilBinding := r.lookupBinding(scope, "nil", statement.Pos())
	if nilBinding == types.Universe.Lookup("nil") {
		return nil, ast.NewIdent("nil")
	}
	results := r.shortReturnResults()
	errorResultType := results.At(results.Len() - 1).Type()
	nilResult := r.Name()
	errorTypeSyntax := r.TemporaryType(errorResultType, statement.Results[0])
	nilDeclaration := Variable(nilResult, errorTypeSyntax, nil)
	return []ast.Stmt{nilDeclaration}, nilResult
}

// shortReturnResults reads the declared results of the function that owns the short return.
func (r *Context) shortReturnResults() *types.Tuple {
	return r.frameSignature().Results()
}

// shortReturnTuple uses declared result types, so private callee types need no name.
func (r *Context) shortReturnTuple(tupleCall, originalCall ast.Expr) ([]ast.Stmt, []ast.Expr) {
	declaredResults := r.shortReturnResults()
	var statements []ast.Stmt
	var resultReferences []ast.Expr
	for resultIndex := 0; resultIndex < declaredResults.Len()-1; resultIndex++ {
		resultName := r.Name()
		resultType := r.TemporaryType(declaredResults.At(resultIndex).Type(), originalCall)
		statements = append(statements, Variable(resultName, resultType, nil))
		resultReferences = append(resultReferences, resultName)
	}
	statements = append(statements, &ast.AssignStmt{Lhs: resultReferences, Tok: token.ASSIGN,
		Rhs: []ast.Expr{tupleCall}})
	return statements, resultReferences
}
