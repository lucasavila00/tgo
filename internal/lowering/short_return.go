package lowering

import (
	"go/ast"

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
		values = append(append([]ast.Expr(nil), r.frame.values...), values...)
	} else {
		if len(values) == 1 && r.resultCount(call.Args[0]) > 1 {
			saved, refs := r.save(values[0])
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
	scope := r.source.checked.Types.Scope().Innermost(statement.Pos())
	_, object := scope.LookupParent("nil", statement.Pos())
	if object == types.Universe.Lookup("nil") {
		return nil, ast.NewIdent("nil")
	}
	var signature *types.Signature
	switch function := r.frame.function.(type) {
	case *ast.FuncDecl:
		signature = r.source.checked.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	case *ast.FuncLit:
		signature = r.source.checked.TypesInfo.TypeOf(function).(*types.Signature)
	}
	last := signature.Results().At(signature.Results().Len() - 1).Type()
	zero := r.name()
	return []ast.Stmt{variable(zero, r.temporaryType(last, statement.Results[0]), nil)}, zero
}
