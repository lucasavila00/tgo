package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// shortReturnStatement supplies omitted results without a function wrapper.
func (r *rewrite) shortReturnStatement(statement *ast.ReturnStmt, record frontend.ShortReturn) []ast.Stmt {
	returnMarker := statement.Results[0].(*ast.CallExpr)
	statementsEvaluation := r.operands(returnMarker.Args)
	statements, returnValues := statementsEvaluation.statements, statementsEvaluation.expressions()
	if record.ZeroSuccess {
		if !r.functionState.frame.prepared {
			r.functionState.frame.prefix = r.prepareReturns(r.functionState.frame.typ)
			r.functionState.frame.prepared = true
		}
		zeroValues := append([]ast.Expr(nil), r.functionState.frame.values...)
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

func (r *rewrite) shortReturnNil(statement *ast.ReturnStmt) ([]ast.Stmt, ast.Expr) {
	scope := r.input.source.inventory.Checked.Types.Scope().Innermost(statement.Pos())
	_, nilBinding := scope.LookupParent("nil", statement.Pos())
	if nilBinding == types.Universe.Lookup("nil") {
		return nil, ast.NewIdent("nil")
	}
	results := r.shortReturnResults()
	errorResultType := results.At(results.Len() - 1).Type()
	nilResult := r.name()
	errorTypeSyntax := r.temporaryType(errorResultType, statement.Results[0])
	nilDeclaration := variable(nilResult, errorTypeSyntax, nil)
	return []ast.Stmt{nilDeclaration}, nilResult
}

func (r *rewrite) shortReturnResults() *types.Tuple {
	return r.frameSignature().Results()
}

// shortReturnTuple uses declared result types, so private callee types need no name.
func (r *rewrite) shortReturnTuple(tupleCall, originalCall ast.Expr) ([]ast.Stmt, []ast.Expr) {
	declaredResults := r.shortReturnResults()
	var statements []ast.Stmt
	var resultReferences []ast.Expr
	for resultIndex := 0; resultIndex < declaredResults.Len()-1; resultIndex++ {
		resultName := r.name()
		resultType := r.temporaryType(declaredResults.At(resultIndex).Type(), originalCall)
		statements = append(statements, variable(resultName, resultType, nil))
		resultReferences = append(resultReferences, resultName)
	}
	statements = append(statements, &ast.AssignStmt{Lhs: resultReferences, Tok: token.ASSIGN, Rhs: []ast.Expr{tupleCall}})
	return statements, resultReferences
}
