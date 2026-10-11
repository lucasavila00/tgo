package types

import (
	"go/ast"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// shortReturn checks the supplied operands against the source result types.
func (check *Checker) shortReturn(statement *ast.ReturnStmt) bool {
	if len(statement.Results) != 1 {
		return false
	}
	call, ok := statement.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	operation, found := check.tgoOperation(call)
	if !found || !operation.ShortReturn {
		return false
	}
	if check.sig == nil {
		check.error(statement, InvalidSyntaxTree, "short return requires an enclosing function")
		return true
	}
	results := check.sig.results
	if results.Len() == 0 {
		check.error(statement, IncompatibleAssign, "short return requires a final result that implements error")
		return true
	}
	for _, result := range results.vars {
		if result.name != "" {
			check.error(statement, InvalidSyntaxTree, "short return requires unnamed results")
			return true
		}
	}
	if !AssignableTo(results.At(results.Len()-1).typ, Universe.Lookup("error").Type()) {
		check.error(statement, IncompatibleAssign, "short return requires a final result that implements error")
		return true
	}
	if call.Ellipsis.IsValid() {
		check.errorf(atPos(call.Ellipsis), InvalidSyntaxTree, "short return operands cannot end with ...")
		return true
	}
	last := results.Len() - 1
	if operation.ZeroSuccess {
		check.initVars(results.vars[last:], call.Args, statement)
	} else {
		if !AssignableTo(Typ[UntypedNil], results.At(last).typ) {
			check.error(statement, IncompatibleAssign, "short return requires nil to fit the final result type")
			return true
		}
		check.initVars(results.vars[:last], call.Args, statement)
	}
	return true
}
