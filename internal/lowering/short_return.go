package lowering

import (
	"go/ast"
	"go/token"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
)

// shortReturnStatement supplies omitted results without a function wrapper.
func (r *rewrite) shortReturnStatement(statement *ast.ReturnStmt, record frontend.ShortReturn) []ast.Stmt {
	call := statement.Results[0].(*ast.CallExpr)
	before, values := r.operands(call.Args)
	if record.ZeroSuccess {
		if !r.frame.prepared {
			r.frame.prefix = r.prepareReturns(r.frame.typ)
			r.frame.prepared = true
		}
		zeroValues := append([]ast.Expr(nil), r.frame.values...)
		values = append(zeroValues, values...)
	} else {
		if len(values) == 1 && r.resultCount(call.Args[0]) > 1 {
			saved, refs := r.shortReturnTuple(values[0], call.Args[0])
			before = append(before, saved...)
			values = refs
		}
		saved, zero := r.shortReturnNil(statement)
		before = append(before, saved...)
		values = append(values, zero)
	}
	copy := *statement
	copy.Results = values
	r.loweredReturns[statement] = true
	return append(before, &copy)
}

func (r *rewrite) shortReturnNil(statement *ast.ReturnStmt) ([]ast.Stmt, ast.Expr) {
	scope := r.source.inventory.Checked.Types.Scope().Innermost(statement.Pos())
	_, object := scope.LookupParent("nil", statement.Pos())
	if object == types.Universe.Lookup("nil") {
		return nil, ast.NewIdent("nil")
	}
	results := r.shortReturnResults()
	last := results.At(results.Len() - 1).Type()
	zero := r.name()
	return []ast.Stmt{variable(zero, r.temporaryType(last, statement.Results[0]), nil)}, zero
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
func (r *rewrite) shortReturnTuple(value, original ast.Expr) ([]ast.Stmt, []ast.Expr) {
	results := r.shortReturnResults()
	var statements []ast.Stmt
	var references []ast.Expr
	for i := 0; i < results.Len()-1; i++ {
		name := r.name()
		resultType := r.temporaryType(results.At(i).Type(), original)
		statements = append(statements, variable(name, resultType, nil))
		references = append(references, name)
	}
	statements = append(statements, &ast.AssignStmt{Lhs: references, Tok: token.ASSIGN, Rhs: []ast.Expr{value}})
	return statements, references
}
