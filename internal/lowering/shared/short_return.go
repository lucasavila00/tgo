package shared

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// shortReturnStatement supplies omitted results without a function wrapper.
func (r *Context) shortReturnStatement(statement *ast.ReturnStmt, record frontend.ShortReturn) []ast.Stmt {
	returnMarker := statement.Results[0].(*ast.CallExpr)
	statementsEvaluation := r.Operands(returnMarker.Args)
	statements, returnValues := statementsEvaluation.Statements, statementsEvaluation.Expressions()
	if record.ZeroSuccess {
		zeroValues := r.PrepareFailureResults()
		returnValues = append(zeroValues, returnValues...)
	} else {
		if len(returnValues) == 1 && r.resultCount(returnMarker.Args[0]) > 1 {
			tupleStatements, tupleResults := r.shortReturnTuple(returnValues[0], returnMarker.Args[0])
			statements = append(statements, tupleStatements...)
			returnValues = tupleResults
		}
		nilStatements, nilResult := r.shortReturnNil(statement)
		statements = append(statements, nilStatements...)
		returnValues = append(returnValues, nilResult)
	}
	returnStatement := *statement
	returnStatement.Results = returnValues
	r.request.loweredReturns[statement] = true
	return append(statements, &returnStatement)
}

// shortReturnNil supplies nil for the omitted error result, using typed zero storage when source hides nil.
func (r *Context) shortReturnNil(statement *ast.ReturnStmt) ([]ast.Stmt, ast.Expr) {
	scope := r.input.Inventory.Checked.Types.Scope().Innermost(statement.Pos())
	_, nilBinding := scope.LookupParent("nil", statement.Pos())
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
