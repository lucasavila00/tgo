// Package types checks Go syntax with TGo conditional values and error propagation.
package types

import "github.com/lucasavila00/tgo/internal/ast"

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

// recordExpressionScope stores the source scope before an expression opens its child scopes.
func (check *Checker) recordExpressionScope(expression ast.Expr) {
	if check.Info == nil {
		return
	}
	if check.ExprScopes == nil {
		check.ExprScopes = make(map[ast.Expr]*Scope)
	}
	check.ExprScopes[expression] = check.scope
}

// queuePackageUsage keeps package-level source locals until closure bodies are checked.
func (check *Checker) queuePackageUsage(scope *Scope) {
	if scope.Parent() != nil && scope.Parent().Parent() == check.pkg.scope {
		check.tgo.packageScopes = append(check.tgo.packageScopes, scope)
	}
}

// checkPackageUsage reports unused TGo locals after all source functions are checked.
func (check *Checker) checkPackageUsage() {
	for _, scope := range check.tgo.packageScopes {
		check.usage(scope)
	}
	check.tgo.packageScopes = nil
}

// recordRuntimeExpression keeps native statements visible to array len, cap, and range checks.
func (check *Checker) recordRuntimeExpression() {
	check.hasCallOrRecv = true
}
