// Package types checks Go syntax with TGo conditional values and error propagation.
package types

import (
	"go/ast"
	"go/token"
)

// TGoOperation records the meaning of a frontend marker call.
type TGoOperation struct {
	Ranges        []*ast.RangeStmt
	Comprehension bool
	Output        string
	Element       bool
	Conditional   bool
	ShortReturn   bool
	ZeroSuccess   bool
	Wrap          bool
	CallName      string
	Position      token.Position
}

func (check *Checker) tgoOperation(call *ast.CallExpr) (TGoOperation, bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return TGoOperation{}, false
	}
	operation, found := check.conf.TGo[id.Name]
	return operation, found
}

// expressionType passes context through operators whose result has that type.
func (check *Checker) expressionType(e ast.Expr, typ Type) {
	if e == nil || typ == nil {
		return
	}
	if check.tgo.expressionTypes == nil {
		check.tgo.expressionTypes = make(map[ast.Expr]Type)
	}
	check.tgo.expressionTypes[e] = typ
	switch e := e.(type) {
	case *ast.ParenExpr:
		check.expressionType(e.X, typ)
	case *ast.UnaryExpr:
		if accepts := unaryOpPredicates[e.Op]; accepts != nil && accepts(typ) {
			check.expressionType(e.X, typ)
		}
	case *ast.BinaryExpr:
		if accepts := binaryOpPredicates[e.Op]; (accepts != nil && accepts(typ)) || (isShift(e.Op) &&
			allInteger(typ)) {
			check.expressionType(e.X, typ)
			if !isShift(e.Op) {
				check.expressionType(e.Y, typ)
			}
		}
	}
}
