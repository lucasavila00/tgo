// Package expressioninsert is a source-to-source experiment for issue 248.
package expressioninsert

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// Insert adds statements at the run-time evaluation point of target.
//
// The caller must provide type information for file. Insert changes file in place.
// It returns false without a change when target is not evaluated at run time.
// Before runs immediately before target. After runs after normal evaluation and
// before the result is used. An after statement does not run when target panics.
// The inserted statements stay in the source function, so return, goto, break,
// continue, defer, and recover keep their Go meaning.
func Insert(file *ast.File, info *types.Info, target ast.Expr, before, after []ast.Stmt) (bool, error) {
	if info == nil || target == nil {
		return false, fmt.Errorf("expression insertion needs type information and a target")
	}
	if !runtimeExpression(file, info, target) {
		return false, nil
	}
	var body *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		var candidate *ast.BlockStmt
		switch function := node.(type) {
		case *ast.FuncDecl:
			candidate = function.Body
		case *ast.FuncLit:
			candidate = function.Body
		}
		if candidate != nil && contains(candidate, target) {
			body = candidate
		}
		return body == nil
	})
	if body != nil {
		lowerer := newLowerer(info, target, before, after, body)
		body.List = lowerer.statements(body.List)
		if !lowerer.inserted {
			if lowerer.unsupportedReason != "" {
				return false, fmt.Errorf("selected expression cannot be lowered: %s", lowerer.unsupportedReason)
			}
			return false, fmt.Errorf("selected expression context is not lowered")
		}
		return true, nil
	}
	return false, fmt.Errorf("selected run-time expression is not in a function declaration")
}

func runtimeExpression(file *ast.File, info *types.Info, expression ast.Expr) bool {
	value, ok := info.Types[expression]
	call, callExpression := expression.(*ast.CallExpr)
	if !ok || (!value.IsValue() && (!callExpression || info.Types[call.Fun].IsType())) {
		return false
	}
	path := nodePath(file, expression)
	for _, ancestor := range path {
		switch node := ancestor.(type) {
		case *ast.CallExpr:
			if info.Types[node].Value != nil && containsAny(node.Args, expression) {
				return false
			}
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok &&
				(selector.Sel.Name == "Sizeof" || selector.Sel.Name == "Alignof" || selector.Sel.Name == "Offsetof") &&
				containsAny(node.Args, expression) {
				return false
			}
		case *ast.RangeStmt:
			if node.Value == nil && contains(node.X, expression) && constantRangeLength(info.TypeOf(node.X)) {
				return false
			}
		}
	}
	return true
}

func nodePath(root ast.Node, target ast.Node) []ast.Node {
	stack := []ast.Node(nil)
	var result []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, node)
		if node == target {
			result = append([]ast.Node(nil), stack...)
			return false
		}
		return result == nil
	})
	return result
}

func constantRangeLength(typ types.Type) bool {
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	_, ok := typ.Underlying().(*types.Array)
	return ok
}

func contains(root ast.Node, target ast.Node) bool {
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
		if node == target {
			found = true
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested && node != root {
			return false
		}
		return !found
	})
	return found
}

type lowerer struct {
	info              *types.Info
	target            ast.Expr
	before            []ast.Stmt
	after             []ast.Stmt
	names             map[string]bool
	inserted          bool
	unsupportedReason string
}

func newLowerer(info *types.Info, target ast.Expr, before, after []ast.Stmt, body *ast.BlockStmt) *lowerer {
	names := make(map[string]bool)
	ast.Inspect(body, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			names[identifier.Name] = true
		}
		return true
	})
	return &lowerer{info: info, target: target, before: before, after: after, names: names}
}

func (l *lowerer) fresh(preferred string) *ast.Ident {
	name := preferred
	for index := 0; l.names[name]; index++ {
		name = fmt.Sprintf("%s%d", preferred, index+1)
	}
	l.names[name] = true
	return ast.NewIdent(name)
}

func cloneStatements(input []ast.Stmt) []ast.Stmt {
	return append([]ast.Stmt(nil), input...)
}

func (l *lowerer) capture(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression != l.target {
		return l.expression(expression)
	}
	l.inserted = true
	prefix := cloneStatements(l.before)
	if l.info.Types[expression].Value != nil {
		prefix = append(prefix, cloneStatements(l.after)...)
		return expression, prefix
	}
	name := l.fresh("expressionValue")
	prefix = append(prefix, &ast.AssignStmt{
		Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
	})
	prefix = append(prefix, cloneStatements(l.after)...)
	return ast.NewIdent(name.Name), prefix
}

func (l *lowerer) expression(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == nil || !contains(expression, l.target) {
		return expression, nil
	}
	if expression == l.target {
		return l.capture(expression)
	}
	switch node := expression.(type) {
	case *ast.ParenExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.SelectorExpr:
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
	case *ast.IndexExpr:
		return l.ordered(node, []ast.Expr{node.X, node.Index}, func(values []ast.Expr) {
			node.X, node.Index = values[0], values[1]
		})
	case *ast.IndexListExpr:
		values := append([]ast.Expr{node.X}, node.Indices...)
		return l.ordered(node, values, func(values []ast.Expr) {
			node.X, node.Indices = values[0], values[1:]
		})
	case *ast.SliceExpr:
		return l.ordered(node, []ast.Expr{node.X, node.Low, node.High, node.Max}, func(values []ast.Expr) {
			node.X, node.Low, node.High, node.Max = values[0], values[1], values[2], values[3]
		})
	case *ast.BinaryExpr:
		if node.Op == token.LAND || node.Op == token.LOR {
			return l.shortCircuit(node)
		}
		return l.ordered(node, []ast.Expr{node.X, node.Y}, func(values []ast.Expr) {
			node.X, node.Y = values[0], values[1]
		})
	case *ast.CallExpr:
		if len(node.Args) == 1 && node.Args[0] == l.target {
			if prefix := l.tupleCapture(node.Args[0], func(values []ast.Expr) { node.Args = values }); prefix != nil {
				return node, prefix
			}
		}
		values := append([]ast.Expr{node.Fun}, node.Args...)
		return l.ordered(node, values, func(values []ast.Expr) {
			node.Fun, node.Args = values[0], values[1:]
		})
	case *ast.CompositeLit:
		return l.ordered(node, node.Elts, func(values []ast.Expr) { node.Elts = values })
	case *ast.KeyValueExpr:
		// A key that names a struct field is not evaluated.
		if contains(node.Value, l.target) {
			value, prefix := l.expression(node.Value)
			node.Value = value
			return node, prefix
		}
	}
	return expression, nil
}

func (l *lowerer) ordered(original ast.Expr, values []ast.Expr, set func([]ast.Expr)) (ast.Expr, []ast.Stmt) {
	prefix := []ast.Stmt(nil)
	result := append([]ast.Expr(nil), values...)
	for index, value := range result {
		if value == nil {
			continue
		}
		lowered, before := l.expression(value)
		if len(before) > 0 {
			for previous := 0; previous < index; previous++ {
				if result[previous] == nil || isStable(result[previous]) {
					continue
				}
				name := l.fresh("expressionOperand")
				prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{result[previous]}})
				result[previous] = ast.NewIdent(name.Name)
			}
		}
		prefix = append(prefix, before...)
		result[index] = lowered
	}
	set(result)
	return original, prefix
}

func isStable(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	default:
		return false
	}
}

func (l *lowerer) shortCircuit(node *ast.BinaryExpr) (ast.Expr, []ast.Stmt) {
	if contains(node.X, l.target) {
		left, prefix := l.expression(node.X)
		node.X = left
		return node, prefix
	}
	left := node.X
	result := l.fresh("expressionCondition")
	initial := "false"
	condition := left
	if node.Op == token.LOR {
		initial = "true"
		condition = &ast.UnaryExpr{Op: token.NOT, X: left}
	}
	right, prefix := l.expression(node.Y)
	body := append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(result.Name)}, Tok: token.ASSIGN, Rhs: []ast.Expr{right}})
	return ast.NewIdent(result.Name), []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{result}, Tok: token.DEFINE, Rhs: []ast.Expr{ast.NewIdent(initial)}},
		&ast.IfStmt{Cond: condition, Body: &ast.BlockStmt{List: body}},
	}
}
