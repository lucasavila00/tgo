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
	setupStatements, returnValues := r.operands(returnMarker.Args)
	if record.ZeroSuccess {
		if !r.frame.prepared {
			r.frame.prefix = r.prepareReturns(r.frame.typ)
			r.frame.prepared = true
		}
		zeroValues := append([]ast.Expr(nil), r.frame.values...)
		returnValues = append(zeroValues, returnValues...)
	} else {
		if len(returnValues) == 1 && r.resultCount(returnMarker.Args[0]) > 1 {
			tupleSetup, tupleResults := r.shortReturnTuple(returnValues[0], returnMarker.Args[0])
			setupStatements = append(setupStatements, tupleSetup...)
			returnValues = tupleResults
		}
		nilSetup, nilResult := r.shortReturnNil(statement)
		setupStatements = append(setupStatements, nilSetup...)
		returnValues = append(returnValues, nilResult)
	}
	returnStatement := *statement
	returnStatement.Results = returnValues
	r.loweredReturns[statement] = true
	return append(setupStatements, &returnStatement)
}

func (r *rewrite) shortReturnNil(statement *ast.ReturnStmt) ([]ast.Stmt, ast.Expr) {
	scope := r.source.inventory.Checked.Types.Scope().Innermost(statement.Pos())
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
	var signature *types.Signature
	switch function := r.frame.function.(type) {
	case *ast.FuncDecl:
		signature = r.source.inventory.Checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.source.inventory.Checked.TypesInfo.TypeOf(function).(*types.Signature)
	}
	return signature.Results()
}

// shortReturnTuple uses declared result types, so private callee types need no name.
func (r *rewrite) shortReturnTuple(tupleCall, originalCall ast.Expr) ([]ast.Stmt, []ast.Expr) {
	declaredResults := r.shortReturnResults()
	var setupStatements []ast.Stmt
	var resultReferences []ast.Expr
	for resultIndex := 0; resultIndex < declaredResults.Len()-1; resultIndex++ {
		resultName := r.name()
		resultType := r.temporaryType(declaredResults.At(resultIndex).Type(), originalCall)
		setupStatements = append(setupStatements, variable(resultName, resultType, nil))
		resultReferences = append(resultReferences, resultName)
	}
	setupStatements = append(setupStatements, &ast.AssignStmt{Lhs: resultReferences, Tok: token.ASSIGN, Rhs: []ast.Expr{tupleCall}})
	return setupStatements, resultReferences
}
