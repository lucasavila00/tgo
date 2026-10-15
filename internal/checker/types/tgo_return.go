package types

import (
	"github.com/lucasavila00/tgo/internal/ast"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// shortReturn checks the supplied operands against the source result types.
func (check *Checker) shortReturn(statement *ast.ShortReturnStmt) {
	if check.sig == nil {
		check.error(statement, InvalidSyntaxTree, "short return requires an enclosing function")
		return
	}
	results := check.sig.results
	if results.Len() == 0 {
		check.error(statement, IncompatibleAssign, "short return requires a final result that implements error")
		return
	}
	for _, result := range results.vars {
		if result.name != "" {
			check.error(statement, InvalidSyntaxTree, "short return requires unnamed results")
			return
		}
	}
	errorResultIndex := results.Len() - 1
	errorResultType := results.At(errorResultIndex).typ
	if !AssignableTo(errorResultType, Universe.Lookup("error").Type()) {
		check.error(statement, IncompatibleAssign, "short return requires a final result that implements error")
		return
	}
	if statement.ZeroSuccess {
		check.initVars(results.vars[errorResultIndex:], statement.Results, statement)
	} else {
		if !AssignableTo(Typ[UntypedNil], errorResultType) {
			check.error(statement, IncompatibleAssign, "short return requires nil to fit the final result type")
			return
		}
		check.initVars(results.vars[:errorResultIndex], statement.Results, statement)
	}
}
