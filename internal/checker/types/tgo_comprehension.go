package types

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"go/token"

	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// comprehensionExpr checks range sources, bindings, filters, and output in execution order.
func (check *Checker) comprehensionExpr(x *operand, comprehension *ast.ComprehensionExpr) exprKind {
	defer check.recordRuntimeExpression()
	typ := check.typ(comprehension.Type)
	key, element := ComprehensionTypes(typ)
	x.expr, x.typ_ = comprehension, Typ[Invalid]
	if element == nil {
		check.error(comprehension.Type, InvalidLit, "comprehension requires a slice or map type with one element type")
		return expression
	}
	previousContext, previousTargets := check.statementContext, check.controlTargets
	check.statementContext &^= fallthroughOk | finalSwitchCase | inTypeSwitch
	defer func() {
		check.statementContext, check.controlTargets = previousContext, previousTargets
	}()
	packageBody := check.sig == nil && check.scope.Parent() == check.pkg.scope
	check.openScope(comprehension, "comprehension")
	defer check.closeScope()
	if packageBody {
		check.queuePackageUsage(check.scope)
	}
	for _, clause := range comprehension.Clauses {
		switch clause := clause.(type) {
		case *ast.RangeClause:
			if clause.Key != nil && clause.Tok != token.DEFINE {
				check.error(clause, InvalidSyntaxTree, "comprehension generators must declare bindings with :=")
			}
			position := clause.TokPos
			if !position.IsValid() {
				position = clause.Range
			}
			check.rangeHeader(clause, token.NoPos, atPos(position), clause.Key, clause.Value, nil, clause.X,
				clause.Tok == token.DEFINE)
			defer check.closeScope()
			check.controlTargets = append(check.controlTargets, clause)
			check.statementContext |= breakOk | continueOk
		case *ast.FilterClause:
			var condition operand
			check.expr(nil, &condition, clause.Cond)
			if condition.isValid() && !allBoolean(condition.typ()) {
				check.error(clause.Cond, InvalidCond, "non-boolean condition in if statement")
			}
		}
	}
	targets := []Type{element}
	if key != nil {
		targets = []Type{key, element}
	}
	outputs := comprehensionOperands(comprehension.Output)
	if len(outputs) != len(targets) {
		check.error(comprehension, WrongArgCount, "comprehension output must match the collection's element or key and value")
		return expression
	}
	for i, expression := range outputs {
		var output operand
		check.rawExpr(newTarget(targets[i], "comprehension output"), &output, expression, targets[i], false)
		check.assignment(&output, targets[i], "comprehension output")
	}
	x.mode_, x.typ_ = value, typ
	return expression
}

// comprehensionOperands reads key:value outputs without changing their source nodes.
func comprehensionOperands(outputs []ast.Expr) []ast.Expr {
	var operands []ast.Expr
	for _, output := range outputs {
		if pair, ok := output.(*ast.KeyValueExpr); ok {
			operands = append(operands, pair.Key, pair.Value)
		} else {
			operands = append(operands, output)
		}
	}
	return operands
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

// isComprehension reports a native collection comprehension.
func (check *Checker) isComprehension(expression ast.Expr) bool {
	_, ok := ast.Unparen(expression).(*ast.ComprehensionExpr)
	return ok
}
