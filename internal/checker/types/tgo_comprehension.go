package types

import (
	"go/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

func (check *Checker) comprehensionExpr(x *operand, call *ast.CallExpr) exprKind {
	operation, _ := check.tgoOperation(call)
	if check.comprehensionRanges == nil {
		check.comprehensionRanges = make(map[*ast.RangeStmt]bool)
	}
	for _, loop := range operation.Ranges {
		check.comprehensionRanges[loop] = true
		if loop.Key != nil && loop.Tok != token.DEFINE {
			check.error(loop, InvalidSyntaxTree, "comprehension generators must declare bindings with :=")
		}
	}
	typ := check.typ(call.Args[0])
	key, element := ComprehensionTypes(typ)
	x.expr, x.typ_ = call, Typ[Invalid]
	if element == nil {
		check.error(call.Args[0], InvalidLit, "comprehension requires a slice or map type with one element type")
		return expression
	}
	if check.comprehensionTypes == nil {
		check.comprehensionTypes = make(map[string][]Type)
	}
	check.comprehensionTypes[operation.Output] = []Type{element}
	if key != nil {
		check.comprehensionTypes[operation.Output] = []Type{key, element}
	}
	previous := check.statementContext
	check.statementContext &^= fallthroughOk | finalSwitchCase | inTypeSwitch
	body := call.Args[1].(*ast.FuncLit).Body
	packageBody := check.sig == nil && check.scope.Parent() == check.pkg.scope
	check.openScope(body, "comprehension")
	check.stmtList(check.statementContext, body.List)
	if packageBody {
		check.usage(check.scope)
	}
	check.closeScope()
	check.statementContext = previous
	check.hasCallOrRecv = true
	x.mode_, x.typ_ = value, typ
	return expression
}

func (check *Checker) comprehensionOutput(x *operand, call *ast.CallExpr) exprKind {
	id := call.Fun.(*ast.Ident)
	targets := check.comprehensionTypes[id.Name]
	x.expr, x.mode_, x.typ_ = call, novalue, NewTuple()
	if len(call.Args) != len(targets) {
		check.error(call, WrongArgCount, "comprehension output must match the collection's element or key and value")
		x.invalidate()
		return statement
	}
	for i, expression := range call.Args {
		var output operand
		check.rawExpr(newTarget(targets[i], "comprehension output"), &output, expression, targets[i], false)
		check.assignment(&output, targets[i], "comprehension output")
	}
	return statement
}

// operationStatements returns source statements without a marker function scope.
func (check *Checker) operationStatements(call *ast.CallExpr) ast.Stmt {
	operation, found := check.tgoOperation(call)
	if !found {
		return nil
	}
	if operation.Conditional {
		return call.Args[0].(*ast.FuncLit).Body.List[0]
	}
	if operation.Comprehension {
		return call.Args[1].(*ast.FuncLit).Body
	}
	return nil
}

// ComprehensionTypes returns the common key and element types of a collection.
// A nil element means the type cannot be used for a comprehension.
func ComprehensionTypes(typ Type) (Type, Type) {
	var key, element Type
	valid := underIs(typ, func(underlying Type) bool {
		var k, e Type
		switch collection := underlying.(type) {
		case *Slice:
			e = collection.elem
		case *Map:
			k, e = collection.key, collection.elem
		default:
			return false
		}
		if element != nil && (!Identical(element, e) || (key == nil) != (k == nil) || (key != nil &&
			!Identical(key, k))) {
			return false
		}
		key, element = k, e
		return true
	})

	if !valid {
		return nil, nil
	}
	return key, element
}

func (check *Checker) isComprehension(expression ast.Expr) bool {
	if call, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
		op, found := check.tgoOperation(call)
		return found && op.Comprehension
	}
	return false
}
