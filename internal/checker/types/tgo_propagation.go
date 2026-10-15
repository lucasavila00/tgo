package types

import (
	"github.com/lucasavila00/tgo/internal/ast"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// propagationExpr checks the call final error and the enclosing function error result.
// It records only the success results as the postfix expression type.
func (check *Checker) propagationExpr(x *operand, call *ast.PropagateExpr) exprKind {
	defer check.recordRuntimeExpression()
	position := atPos(call.OpPos)
	if propagationCall(call.X) == nil {
		check.error(position, InvalidSyntaxTree, "propagation requires a call")
		x.invalidate()
		return expression
	}
	check.rawExpr(nil, x, call.X, nil, false)
	x.expr = call
	if !x.isValid() {
		return expression
	}
	results, hasTupleResults := x.typ().(*Tuple)
	errorResultType := x.typ()
	if hasTupleResults && results.Len() != 0 {
		errorResultType = results.At(results.Len() - 1).Type()
	}
	if !AssignableTo(errorResultType, Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error result")
		x.invalidate()
		return expression
	}
	if check.sig == nil || check.sig.Results().Len() == 0 {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	returnedErrorType := errorResultType
	operator := "!"
	if call.Wrap {
		returnedErrorType = Universe.Lookup("error").Type()
		operator = "!!"
	}
	returnErrorType := check.sig.Results().At(check.sig.Results().Len() - 1).Type()
	if _, ok := returnErrorType.Underlying().(*Interface); !ok || !AssignableTo(returnErrorType,
		Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	if !AssignableTo(returnedErrorType, returnErrorType) {
		check.errorf(position, IncompatibleAssign, "%s cannot return error of type %s as %s", operator,
			returnedErrorType, returnErrorType)
		x.invalidate()
		return expression
	}
	x.mode_ = value
	switch {
	case !hasTupleResults || results.Len() == 1:
		x.mode_ = novalue
		x.typ_ = NewTuple()
	case results.Len() == 2:
		x.typ_ = results.At(0).Type()
	default:
		x.typ_ = NewTuple(results.vars[:results.Len()-1]...)
	}
	return statement
}

// propagationCall finds the source call through parentheses and earlier propagation.
func propagationCall(expression ast.Expr) *ast.CallExpr {
	for {
		switch node := expression.(type) {
		case *ast.ParenExpr:
			expression = node.X
		case *ast.PropagateExpr:
			expression = node.X
		case *ast.CallExpr:
			return node
		default:
			return nil
		}
	}
}

// tupleExpression reports source expressions that can supply several results.
func tupleExpression(expression ast.Expr) bool {
	switch ast.Unparen(expression).(type) {
	case *ast.CallExpr, *ast.PropagateExpr:
		return true
	}
	return false
}
