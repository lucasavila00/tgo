package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
)

// enumSwitchStatement snapshots the input and gives each case its own copied binding.
func (r *Context) enumSwitchStatement(
	statement *ast.TypeSwitchStmt, enum *types.Enum, result *ast.Ident, transform ValueBlock,
) []ast.Stmt {
	exit := r.prepareBreakExit(statement, statement.Init, statement.Assign)
	var before []ast.Stmt
	if statement.Init != nil {
		before = r.Header(statement.Init)
	}
	assertion := enumSwitchAssertion(statement.Assign)
	evaluation := r.Expression(assertion.X)
	before = append(before, evaluation.Statements...)
	saved := r.Save(evaluation.Values[0])
	before = append(before, saved.Statements...)
	snapshot := saved.Expressions()[0]
	var payloadInput ast.Expr = snapshot
	if _, parameter := r.TypeOf(assertion.X).(*types.TypeParam); parameter {
		enumType := r.TemporaryType(enum.Named(), assertion.X)
		payloadInput = &ast.CallExpr{Fun: enumType, Args: []ast.Expr{snapshot}}
	}
	output := &ast.SwitchStmt{Switch: statement.Switch,
		Tag:  &ast.CallExpr{Fun: &ast.SelectorExpr{X: payloadInput, Sel: ast.NewIdent("Tag")}},
		Body: &ast.BlockStmt{}}
	if r.functionState.controlTargets == nil {
		r.functionState.controlTargets = make(map[ast.Node]ast.Stmt)
	}
	r.functionState.controlTargets[statement] = output
	exhaustive, hasDefault := false, false
	for _, node := range statement.Body.List {
		clause := node.(*ast.CaseClause)
		if clause.Exhaustive {
			exhaustive = true
			continue
		}
		hasDefault = hasDefault || clause.List == nil
		output.Body.List = append(output.Body.List, r.enumSwitchCase(clause, enum, snapshot, payloadInput, result, transform))
	}
	if exhaustive && !hasDefault {
		last := output.Body.List[len(output.Body.List)-1].(*ast.CaseClause)
		last.List = nil
	}
	body := &ast.BlockStmt{List: append(before, output)}
	return r.withBreakExit(statement, output, []ast.Stmt{body}, exit)
}

// enumSwitchCase replaces source variant names with tags and creates the checked case binding.
func (r *Context) enumSwitchCase(
	clause *ast.CaseClause, enum *types.Enum, snapshot, payloadInput ast.Expr, result *ast.Ident, transform ValueBlock,
) *ast.CaseClause {
	output := *clause
	output.List = nil
	var selected *types.EnumVariant
	for index, expression := range clause.List {
		variant := r.input.Inventory.Checked.TypesInfo.EnumCases[clause][index]
		tagName := enum.Named().Obj().Name() + "Tag" + variant.Name()
		output.List = append(output.List, r.enumMember(variant, expression, expression, tagName))
		selected = variant
	}

	var prefix []ast.Stmt
	binding := r.input.Inventory.Checked.TypesInfo.Implicits[clause]
	if binding != nil && r.enumBindingUsed(clause, binding) {
		name := r.NameFor(binding.Name())
		if r.file.bindings == nil {
			r.file.bindings = make(map[types.Object]string)
		}
		r.file.bindings[binding] = name.Name
		value := snapshot
		var typeName ast.Expr
		if len(clause.List) == 1 {
			value = &ast.CallExpr{Fun: &ast.SelectorExpr{X: payloadInput, Sel: ast.NewIdent(selected.Name() + "Payload")}}
			typeName = r.enumMember(selected, clause.List[0], clause.List[0], selected.Type().Obj().Name())
		} else {
			typeName = r.TemporaryType(binding.Type(), enumSwitchSource(clause))
		}
		prefix = append(prefix, Variable(name, typeName, value))
	}
	output.Body = append(prefix, r.switchCaseBody(clause, result, transform)...)
	return &output
}

// enumBindingUsed checks case-local object identity, including uses in closures.
func (r *Context) enumBindingUsed(clause *ast.CaseClause, object types.Object) bool {
	used := false
	for _, statement := range clause.Body {
		ast.Inspect(statement, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok && r.input.Inventory.Checked.TypesInfo.Uses[identifier] == object {
				used = true
			}
			return !used
		})
	}
	return used
}

// enumSwitchAssertion returns the checked type-switch input expression.
func enumSwitchAssertion(guard ast.Stmt) *ast.TypeAssertExpr {
	if assignment, ok := guard.(*ast.AssignStmt); ok {
		return assignment.Rhs[0].(*ast.TypeAssertExpr)
	}
	return guard.(*ast.ExprStmt).X.(*ast.TypeAssertExpr)
}

// enumSwitchSource provides a real source expression for typed binding diagnostics.
func enumSwitchSource(clause *ast.CaseClause) ast.Expr {
	if len(clause.List) != 0 {
		return clause.List[0]
	}
	return firstExpression(clause)
}
