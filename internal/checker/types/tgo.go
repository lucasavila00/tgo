// Package types checks Go syntax with TGo error propagation.
package types

import (
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
	"go/ast"
	"go/token"
)

// TGoOperation records the meaning of a frontend marker call.
type TGoOperation struct {
	Wrap     bool
	CallName string
	Position token.Position
}

func (check *Checker) tgoOperation(call *ast.CallExpr) (TGoOperation, bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return TGoOperation{}, false
	}
	op, found := check.conf.TGo[id.Name]
	return op, found
}

func (check *Checker) propagationExpr(x *operand, call *ast.CallExpr, operation TGoOperation) exprKind {
	position := atPos(check.fset.File(call.Pos()).Pos(operation.Position.Offset))
	check.rawExpr(nil, x, call.Args[0], nil, false)
	x.expr = call
	if !x.isValid() {
		return expression
	}
	results, ok := x.typ().(*Tuple)
	last := x.typ()
	if ok && results.Len() != 0 {
		last = results.At(results.Len() - 1).Type()
	}
	if !AssignableTo(last, Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error result")
		x.invalidate()
		return expression
	}
	if check.sig == nil || check.sig.Results().Len() == 0 {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	returned := last
	operator := "!"
	if operation.Wrap {
		returned = Universe.Lookup("error").Type()
		operator = "!!"
	}
	target := check.sig.Results().At(check.sig.Results().Len() - 1).Type()
	if _, ok := target.Underlying().(*Interface); !ok || !AssignableTo(target, Universe.Lookup("error").Type()) {
		check.error(position, IncompatibleAssign, "propagation requires a final error interface return")
		x.invalidate()
		return expression
	}
	if !AssignableTo(returned, target) {
		check.errorf(position, IncompatibleAssign, "%s cannot return error of type %s as %s", operator, returned, target)
		x.invalidate()
		return expression
	}
	x.mode_ = value
	switch {
	case !ok || results.Len() == 1:
		x.mode_ = novalue
		x.typ_ = NewTuple()
	case results.Len() == 2:
		x.typ_ = results.At(0).Type()
	default:
		x.typ_ = NewTuple(results.vars[:results.Len()-1]...)
	}
	return statement
}
