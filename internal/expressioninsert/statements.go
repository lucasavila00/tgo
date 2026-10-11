package expressioninsert

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

func (l *lowerer) statements(input []ast.Stmt) []ast.Stmt {
	output := make([]ast.Stmt, 0, len(input))
	for _, statement := range input {
		output = append(output, l.statement(statement)...)
	}
	return output
}

// statement changes only the path that contains the selected expression.
func (l *lowerer) statement(statement ast.Stmt) []ast.Stmt { //nolint:cyclop
	if !contains(statement, l.target) {
		return []ast.Stmt{statement}
	}
	switch node := statement.(type) {
	case *ast.BlockStmt:
		node.List = l.statements(node.List)
		return []ast.Stmt{node}
	case *ast.ExprStmt:
		if node.X == l.target && !l.info.Types[node.X].IsValue() {
			l.inserted = true
			result := cloneStatements(l.before)
			result = append(result, node)
			return append(result, cloneStatements(l.after)...)
		}
		value, prefix := l.expression(node.X)
		node.X = value
		return append(prefix, node)
	case *ast.ReturnStmt:
		if len(node.Results) == 1 && node.Results[0] == l.target {
			if prefix := l.tupleCapture(node.Results[0], func(values []ast.Expr) { node.Results = values }); prefix != nil {
				return append(prefix, node)
			}
		}
		values, prefix := l.expressionList(node.Results)
		node.Results = values
		return append(prefix, node)
	case *ast.AssignStmt:
		if containsAny(node.Lhs, l.target) {
			left, prefix := l.locations(node.Lhs)
			node.Lhs = left
			return append(prefix, node)
		}
		if len(node.Rhs) == 1 && node.Rhs[0] == l.target {
			if prefix := l.tupleCapture(node.Rhs[0], func(values []ast.Expr) { node.Rhs = values }); prefix != nil {
				left, leftPrefix := l.freezeLocations(node.Lhs)
				node.Lhs = left
				return append(append(leftPrefix, prefix...), node)
			}
		}
		values, prefix := l.expressionList(node.Rhs)
		node.Rhs = values
		if len(prefix) == 0 {
			return []ast.Stmt{node}
		}
		left, leftPrefix := l.freezeLocations(node.Lhs)
		node.Lhs = left
		return append(append(leftPrefix, prefix...), node)
	case *ast.IncDecStmt:
		value, prefix := l.location(node.X)
		node.X = value
		return append(prefix, node)
	case *ast.SendStmt:
		values, prefix := l.expressionList([]ast.Expr{node.Chan, node.Value})
		node.Chan, node.Value = values[0], values[1]
		return append(prefix, node)
	case *ast.IfStmt:
		return l.ifStatement(node)
	case *ast.ForStmt:
		return l.forStatement(node)
	case *ast.RangeStmt:
		value, prefix := l.expression(node.X)
		node.X = value
		node.Body.List = l.statements(node.Body.List)
		return append(prefix, node)
	case *ast.SwitchStmt:
		return l.switchStatement(node)
	case *ast.TypeSwitchStmt:
		node.Body.List = l.caseClauses(node.Body.List)
		return []ast.Stmt{node}
	case *ast.SelectStmt:
		return l.selectStatement(node)
	case *ast.GoStmt:
		if node.Call == l.target {
			return l.unsupported(statement, "a root go call runs in a new goroutine")
		}
		value, prefix := l.expression(node.Call)
		node.Call = value.(*ast.CallExpr)
		return append(prefix, node)
	case *ast.DeferStmt:
		if node.Call == l.target {
			return l.unsupported(statement, "a root deferred call cannot be surrounded without a new frame")
		}
		value, prefix := l.expression(node.Call)
		node.Call = value.(*ast.CallExpr)
		return append(prefix, node)
	case *ast.LabeledStmt:
		rewritten := l.statement(node.Stmt)
		if len(rewritten) == 1 {
			node.Stmt = rewritten[0]
			return []ast.Stmt{node}
		}
		node.Stmt = &ast.BlockStmt{List: rewritten}
		return []ast.Stmt{node}
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok {
			return []ast.Stmt{node}
		}
		prefix := []ast.Stmt(nil)
		for _, specification := range declaration.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || !containsAny(value.Values, l.target) {
				continue
			}
			value.Values, prefix = l.expressionList(value.Values)
		}
		return append(prefix, node)
	default:
		return l.unsupported(statement, fmt.Sprintf("statement %T", statement))
	}
}

func (l *lowerer) unsupported(statement ast.Stmt, reason string) []ast.Stmt {
	// Leave the node unchanged. Insert reports that no insertion was made.
	l.unsupportedReason = reason
	return []ast.Stmt{statement}
}

func (l *lowerer) tupleCapture(expression ast.Expr, set func([]ast.Expr)) []ast.Stmt {
	tuple, ok := l.info.TypeOf(expression).(*types.Tuple)
	if !ok {
		return nil
	}
	l.inserted = true
	names := make([]ast.Expr, tuple.Len())
	for index := range tuple.Len() {
		names[index] = l.fresh("expressionValue")
	}
	prefix := cloneStatements(l.before)
	prefix = append(prefix, &ast.AssignStmt{Lhs: names, Tok: token.DEFINE, Rhs: []ast.Expr{expression}})
	prefix = append(prefix, cloneStatements(l.after)...)
	set(identifiers(names))
	return prefix
}

func containsAny(expressions []ast.Expr, target ast.Expr) bool {
	for _, expression := range expressions {
		if contains(expression, target) {
			return true
		}
	}
	return false
}

func (l *lowerer) expressionList(input []ast.Expr) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), input...)
	prefix := []ast.Stmt(nil)
	for index, expression := range result {
		value, before := l.expression(expression)
		if len(before) > 0 {
			for previous := 0; previous < index; previous++ {
				if isStable(result[previous]) {
					continue
				}
				name := l.fresh("expressionOperand")
				prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{result[previous]}})
				result[previous] = ast.NewIdent(name.Name)
			}
		}
		prefix = append(prefix, before...)
		result[index] = value
	}
	return result, prefix
}

func (l *lowerer) locations(input []ast.Expr) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), input...)
	prefix := []ast.Stmt(nil)
	for index, expression := range result {
		value, before := l.location(expression)
		result[index] = value
		prefix = append(prefix, before...)
	}
	return result, prefix
}

func (l *lowerer) freezeLocations(input []ast.Expr) ([]ast.Expr, []ast.Stmt) {
	result := append([]ast.Expr(nil), input...)
	prefix := []ast.Stmt(nil)
	for index, expression := range result {
		value, before := l.freezeLocation(expression)
		result[index] = value
		prefix = append(prefix, before...)
	}
	return result, prefix
}

func (l *lowerer) location(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if expression == l.target {
		l.inserted = true
		prefix := cloneStatements(l.before)
		value, evaluation := l.freezeLocation(expression)
		prefix = append(prefix, evaluation...)
		prefix = append(prefix, cloneStatements(l.after)...)
		return value, prefix
	}
	switch node := expression.(type) {
	case *ast.ParenExpr:
		value, prefix := l.location(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.SelectorExpr:
		value, prefix := l.expression(node.X)
		node.X = value
		return node, prefix
	case *ast.IndexExpr:
		values, prefix := l.expressionList([]ast.Expr{node.X, node.Index})
		node.X, node.Index = values[0], values[1]
		if node == l.target {
			l.inserted = true
			prefix = append(cloneStatements(l.before), prefix...)
			prefix = append(prefix, cloneStatements(l.after)...)
		}
		return node, prefix
	default:
		return expression, nil
	}
}

func (l *lowerer) freezeLocation(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	switch node := expression.(type) {
	case *ast.Ident:
		return node, nil
	case *ast.ParenExpr:
		value, prefix := l.freezeLocation(node.X)
		node.X = value
		return node, prefix
	case *ast.StarExpr:
		value, prefix := l.freezeValue(node.X)
		node.X = value
		return node, prefix
	case *ast.SelectorExpr:
		var value ast.Expr
		var prefix []ast.Stmt
		if isPointerType(l.info.TypeOf(node.X)) {
			value, prefix = l.freezeValue(node.X)
		} else {
			value, prefix = l.freezeLocation(node.X)
		}
		node.X = value
		return node, prefix
	case *ast.IndexExpr:
		var value ast.Expr
		var prefix []ast.Stmt
		if hasArrayType(l.info.TypeOf(node.X)) {
			value, prefix = l.freezeLocation(node.X)
		} else {
			value, prefix = l.freezeValue(node.X)
		}
		index, indexPrefix := l.freezeValue(node.Index)
		node.X, node.Index = value, index
		return node, append(prefix, indexPrefix...)
	default:
		return expression, nil
	}
}

func (l *lowerer) freezeValue(expression ast.Expr) (ast.Expr, []ast.Stmt) {
	if _, literal := expression.(*ast.BasicLit); literal {
		return expression, nil
	}
	name := l.fresh("expressionOperand")
	return ast.NewIdent(name.Name), []ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{name}, Tok: token.DEFINE, Rhs: []ast.Expr{expression},
	}}
}

func hasArrayType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	_, ok := typ.Underlying().(*types.Array)
	return ok
}

func isPointerType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	_, ok := types.Unalias(typ).Underlying().(*types.Pointer)
	return ok
}

func (l *lowerer) ifStatement(node *ast.IfStmt) []ast.Stmt {
	if node.Init != nil && contains(node.Init, l.target) {
		node.Init = oneStatement(l.statement(node.Init))
	}
	node.Body.List = l.statements(node.Body.List)
	if node.Else != nil && contains(node.Else, l.target) {
		node.Else = oneStatement(l.statement(node.Else))
	}
	condition, prefix := l.expression(node.Cond)
	node.Cond = condition
	if len(prefix) > 0 && node.Init != nil {
		prefix = append(l.statement(node.Init), prefix...)
		node.Init = nil
	}
	return append(prefix, node)
}

func oneStatement(input []ast.Stmt) ast.Stmt {
	if len(input) == 1 {
		return input[0]
	}
	return &ast.BlockStmt{List: input}
}

func (l *lowerer) forStatement(node *ast.ForStmt) []ast.Stmt {
	prefix := []ast.Stmt(nil)
	if node.Init != nil && contains(node.Init, l.target) {
		prefix = l.statement(node.Init)
		node.Init = nil
	}
	node.Body.List = l.statements(node.Body.List)
	postPrefix := []ast.Stmt(nil)
	if node.Post != nil && contains(node.Post, l.target) {
		pending := l.fresh("expressionPost")
		prefix = append(prefix, &ast.AssignStmt{Lhs: []ast.Expr{pending}, Tok: token.DEFINE, Rhs: []ast.Expr{ast.NewIdent("false")}})
		postWork := l.statement(node.Post)
		postPrefix = []ast.Stmt{&ast.IfStmt{
			Cond: ast.NewIdent(pending.Name),
			Body: &ast.BlockStmt{List: append([]ast.Stmt{&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent("false")},
			}}, postWork...)},
		}}
		node.Post = &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(pending.Name)}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent("true")}}
	}
	condition, conditionPrefix := l.expression(node.Cond)
	if len(postPrefix) > 0 || len(conditionPrefix) > 0 {
		body := append(postPrefix, conditionPrefix...)
		if condition != nil {
			body = append(body, &ast.IfStmt{Cond: &ast.UnaryExpr{Op: token.NOT, X: condition}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}}})
		}
		body = append(body, node.Body.List...)
		node.Body.List = body
		node.Cond = nil
	}
	return append(prefix, node)
}

func (l *lowerer) switchStatement(node *ast.SwitchStmt) []ast.Stmt {
	prefix := []ast.Stmt(nil)
	if node.Init != nil && contains(node.Init, l.target) {
		prefix = l.statement(node.Init)
		node.Init = nil
	}
	if contains(node.Tag, l.target) {
		node.Tag, prefix = l.expression(node.Tag)
	}
	node.Body.List = l.caseClauses(node.Body.List)
	return append(prefix, node)
}

func (l *lowerer) caseClauses(input []ast.Stmt) []ast.Stmt {
	for _, item := range input {
		clause := item.(*ast.CaseClause)
		clause.Body = l.statements(clause.Body)
	}
	return input
}

func (l *lowerer) selectStatement(node *ast.SelectStmt) []ast.Stmt {
	for _, item := range node.Body.List {
		clause := item.(*ast.CommClause)
		if clause.Comm != nil && contains(clause.Comm, l.target) {
			assignment, ok := clause.Comm.(*ast.AssignStmt)
			if !ok || !containsAny(assignment.Lhs, l.target) {
				return l.unsupported(node, "select entry operands need whole-select ordering")
			}
			temporary := make([]ast.Expr, len(assignment.Lhs))
			for index := range temporary {
				temporary[index] = l.fresh("expressionReceive")
			}
			sourceLeft := assignment.Lhs
			assignment.Lhs = temporary
			assignment.Tok = token.DEFINE
			selectedAssignment := &ast.AssignStmt{Lhs: sourceLeft, Tok: token.ASSIGN, Rhs: identifiers(temporary)}
			clause.Body = append([]ast.Stmt{selectedAssignment}, clause.Body...)
		}
		clause.Body = l.statements(clause.Body)
	}
	return []ast.Stmt{node}
}

func identifiers(input []ast.Expr) []ast.Expr {
	result := make([]ast.Expr, len(input))
	for index, expression := range input {
		result[index] = ast.NewIdent(expression.(*ast.Ident).Name)
	}
	return result
}
