package compiler

import (
	"go/ast"
	"go/token"
)

func (l *propagationLowerer) expressions(input []ast.Expr) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), input...)
	prefix := []ast.Stmt(nil)
	for index, expression := range result {
		lowered, before := l.expression(expression)
		prefix = append(prefix, before...)
		if l.laterLowering(result[index+1:]) {
			lowered, before = l.materializeOrderedOperand(lowered)
			prefix = append(prefix, before...)
		}
		result[index] = lowered
	}
	return result, prefix
}

func (l *propagationLowerer) laterLowering(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		if l.hasLowering(expression) {
			return true
		}
	}
	return false
}

//nolint:cyclop,gocognit // Each expression case keeps Go operand order.
func (l *propagationLowerer) expression(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil || !l.hasLowering(expression) {
		return expression, nil
	}
	if metadata, function, ok := comprehensionMarker(l.source, expression); ok {
		return l.comprehension(metadata, function, expression)
	}
	if metadata, ok := propagationMarker(l.source, expression); ok {
		values, prefix := l.propagation(expression)
		if len(values) != 1 {
			l.unit.failAt(metadata.Bang, "nested propagated call needs one value")
			return expression, prefix
		}
		return values[0], prefix
	}
	switch node := expression.(type) {
	case *ast.CallExpr:
		ordered := append([]ast.Expr{node.Fun}, node.Args...)
		ordered, prefix := l.expressions(ordered)
		node.Fun, node.Args = ordered[0], ordered[1:]
		return node, prefix
	case *ast.BinaryExpr:
		if node.Op == token.LAND || node.Op == token.LOR {
			return l.shortCircuit(node)
		}
		values, prefix := l.expressions([]ast.Expr{node.X, node.Y})
		node.X, node.Y = values[0], values[1]
		return node, prefix
	case *ast.IndexExpr:
		return l.indexExpression(node)
	case *ast.IndexListExpr:
		values := append([]ast.Expr{node.X}, node.Indices...)
		values, prefix := l.expressions(values)
		node.X, node.Indices = values[0], values[1:]
		return node, prefix
	case *ast.SelectorExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.SliceExpr:
		return l.sliceExpression(node)
	case *ast.ParenExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.UnaryExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.TypeAssertExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.CompositeLit:
		values, prefix := l.expressions(node.Elts)
		node.Elts = values
		return node, prefix
	case *ast.KeyValueExpr:
		values, prefix := l.expressions([]ast.Expr{node.Key, node.Value})
		node.Key, node.Value = values[0], values[1]
		return node, prefix
	case *ast.FuncLit:
		return node, nil
	default:
		l.rejectExpression(expression, "this expression")
		return expression, nil
	}
}

// indexExpression keeps an array path addressable while it lowers the index.
func (l *propagationLowerer) indexExpression(
	node *ast.IndexExpr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.indexedOperand(node.X)
	index, indexPrefix := l.expression(node.Index)
	node.X, node.Index = value, index
	return node, append(prefix, indexPrefix...)
}

// sliceExpression keeps an array path addressable while it lowers each bound.
func (l *propagationLowerer) sliceExpression(
	node *ast.SliceExpr,
) (ast.Expr, []ast.Stmt) {
	value, prefix := l.indexedOperand(node.X)
	bounds, boundsPrefix := l.expressions([]ast.Expr{node.Low, node.High, node.Max})
	node.X, node.Low, node.High, node.Max = value, bounds[0], bounds[1], bounds[2]
	return node, append(prefix, boundsPrefix...)
}

// indexedOperand avoids an array copy but saves other computed containers.
func (l *propagationLowerer) indexedOperand(
	expression ast.Expr,
) (ast.Expr, []ast.Stmt) {
	if !admitsArray(l.unit.info.TypeOf(expression)) {
		return l.cheapAssignmentOperand(expression)
	}
	switch expression.(type) {
	case *ast.Ident, *ast.ParenExpr, *ast.StarExpr,
		*ast.SelectorExpr, *ast.IndexExpr:
		return l.assignmentTarget(expression)
	default:
		return l.cheapAssignmentOperand(expression)
	}
}

// shortCircuit keeps conditional right-side evaluation around propagation branches.
func (l *propagationLowerer) shortCircuit(node *ast.BinaryExpr) (ast.Expr, []ast.Stmt) {
	left, prefix := l.expression(node.X)
	defaultValue := "false"
	condition := left
	if node.Op == token.LOR {
		defaultValue = "true"
		condition = &ast.UnaryExpr{Op: token.NOT, X: left}
	}
	resultName := l.freshName("condition")
	prefix = append(prefix, &ast.AssignStmt{
		Lhs: []ast.Expr{resultName},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{l.unit.generatedUniverse(defaultValue, node.OpPos)},
	})
	right, rightPrefix := l.expression(node.Y)
	rightPrefix = append(rightPrefix, &ast.AssignStmt{
		Lhs: []ast.Expr{resultName}, Tok: token.ASSIGN, Rhs: []ast.Expr{right},
	})
	prefix = append(prefix, &ast.IfStmt{
		Cond: condition,
		Body: &ast.BlockStmt{List: rightPrefix},
	})
	return resultName, prefix
}

func (l *propagationLowerer) optionalExpression(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil {
		return nil, nil
	}
	return l.expression(expression)
}

func (l *propagationLowerer) hasLowering(expression ast.Expr) bool {
	if expression == nil {
		return false
	}
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested && node != expression {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if _, ok := propagationMarker(l.source, call); ok {
			found = true
			return false
		}
		if _, _, ok := comprehensionMarker(l.source, call); ok {
			found = true
			return false
		}
		return true
	})
	return found
}
