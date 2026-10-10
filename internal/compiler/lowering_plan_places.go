package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *loweringPlanBuilder) assignmentPlaces(expressions ...ast.Expr) []*plannedPlace {
	result := make([]*plannedPlace, 0, len(expressions))
	for _, expression := range expressions {
		if expression == nil {
			continue
		}
		result = append(result, b.assignmentPlace(expression))
	}
	return result
}

func (b *loweringPlanBuilder) assignmentPlace(expression ast.Expr) *plannedPlace {
	if parenthesized, ok := expression.(*ast.ParenExpr); ok {
		return b.assignmentPlace(parenthesized.X)
	}
	place := &plannedPlace{
		id: placeID(len(b.plan.places) + 1), typ: b.expressionType(expression),
		position: expression.Pos(), source: expression, kind: planObjectPlace,
	}
	b.plan.places = append(b.plan.places, place)
	switch node := expression.(type) {
	case *ast.StarExpr:
		place.kind = planDerefPlace
		place.container = b.expression(node.X)
		place.values = append(place.values, b.newValue(place.container.typ, node.X.Pos()))
	case *ast.SelectorExpr:
		place.kind = planFieldPlace
		if selection := b.unit.info.Selections[node]; selection != nil &&
			len(selection.Index()) > 1 {
			place.base = b.promotedSelectionBase(node, selection)
		} else if _, pointer := types.Unalias(b.expressionType(node.X)).(*types.Pointer); pointer {
			place.base = b.derefPlace(node.X)
		} else {
			place.base = b.assignmentPlace(node.X)
		}
	case *ast.IndexExpr:
		containerPlan := b.expression(node.X)
		containerType := containerPlan.typ
		if !validPlannedType(containerType) && len(containerPlan.results) == 1 {
			containerType = containerPlan.results[0].typ
		}
		containerType = types.Unalias(containerType)
		if named, ok := containerType.(*types.Named); ok {
			containerType = named.Underlying()
		}
		switch containerType.(type) {
		case *types.Array:
			place.kind = planArrayIndexPlace
			place.base = b.assignmentPlace(node.X)
		case *types.Pointer:
			place.kind = planArrayIndexPlace
			place.base = b.derefPlace(node.X)
		case *types.Slice:
			place.kind = planSliceIndexPlace
			place.container = containerPlan
			place.values = append(place.values, b.newValue(place.container.typ, node.X.Pos()))
		case *types.Map:
			place.kind = planMapIndexPlace
			place.container = containerPlan
			place.values = append(place.values, b.newValue(place.container.typ, node.X.Pos()))
		}
		place.index = b.expression(node.Index)
		place.values = append(place.values, b.newValue(place.index.typ, node.Index.Pos()))
	}
	return place
}

func (b *loweringPlanBuilder) promotedSelectionBase(
	node *ast.SelectorExpr,
	selection *types.Selection,
) *plannedPlace {
	base := b.assignmentPlace(node.X)
	expression := node.X
	expressionPlan := b.expression(node.X)
	typ := selection.Recv()
	for _, fieldIndex := range selection.Index()[:len(selection.Index())-1] {
		if pointer, ok := types.Unalias(typ).(*types.Pointer); ok {
			typ = pointer.Elem()
		}
		named := types.Unalias(typ)
		if item, ok := named.(*types.Named); ok {
			named = item.Underlying()
		}
		structure, ok := named.(*types.Struct)
		if !ok || fieldIndex >= structure.NumFields() {
			return base
		}
		field := structure.Field(fieldIndex)
		expression = &ast.SelectorExpr{X: expression, Sel: ast.NewIdent(field.Name())}
		selectorPlan := &plannedExpression{
			kind: planSelectorExpression, source: expression, typ: field.Type(),
			operands: []*plannedExpression{expressionPlan}, resultCount: 1,
		}
		fieldPlace := &plannedPlace{
			id: placeID(len(b.plan.places) + 1), typ: field.Type(),
			position: node.Pos(), kind: planFieldPlace, source: expression, base: base,
		}
		b.plan.places = append(b.plan.places, fieldPlace)
		if _, pointer := types.Unalias(field.Type()).(*types.Pointer); pointer {
			value := b.newValue(field.Type(), node.Pos())
			base = &plannedPlace{
				id: placeID(len(b.plan.places) + 1), typ: dereferencedType(field.Type()),
				position: node.Pos(), kind: planDerefPlace, source: expression,
				base: fieldPlace,
			}
			base.values = append(base.values, value)
			b.plan.places = append(b.plan.places, base)
			expressionPlan = &plannedExpression{
				kind: planRetainedExpression, source: expression, typ: field.Type(),
				materialized: value.id, resultCount: 1,
			}
		} else {
			base = fieldPlace
			expressionPlan = selectorPlan
		}
		typ = field.Type()
	}
	return base
}

func dereferencedType(typ types.Type) types.Type {
	pointer, _ := types.Unalias(typ).(*types.Pointer)
	if pointer == nil {
		return nil
	}
	return pointer.Elem()
}

func (b *loweringPlanBuilder) derefPlace(expression ast.Expr) *plannedPlace {
	pointerType := b.expressionType(expression)
	typ := types.Type(nil)
	if pointer, ok := types.Unalias(pointerType).(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	place := &plannedPlace{
		id: placeID(len(b.plan.places) + 1), typ: typ, position: expression.Pos(),
		kind: planDerefPlace, source: expression, container: b.expression(expression),
	}
	place.values = append(place.values, b.newValue(pointerType, expression.Pos()))
	b.plan.places = append(b.plan.places, place)
	return place
}

// recordBindings carries types from propagation results by source object.
func (b *loweringPlanBuilder) recordBindings(statement ast.Stmt) {
	switch node := statement.(type) {
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE {
			return
		}
		b.bindExpressions(node.Lhs, node.Rhs)
	case *ast.DeclStmt:
		declaration, ok := node.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR {
			return
		}
		for _, specification := range declaration.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			targets := make([]ast.Expr, len(value.Names))
			for index, name := range value.Names {
				targets[index] = name
			}
			b.bindExpressions(targets, value.Values)
		}
	}
}

func (b *loweringPlanBuilder) bindExpressions(targets []ast.Expr, values []ast.Expr) {
	if len(values) == 1 {
		planned := b.expression(values[0])
		if len(planned.results) == len(targets) {
			for index, target := range targets {
				b.bindTarget(target, planned.results[index].typ)
			}
			return
		}
	}
	if len(values) != len(targets) {
		return
	}
	for index, target := range targets {
		b.bindTarget(target, b.expressionType(values[index]))
	}
}

func (b *loweringPlanBuilder) bindTarget(target ast.Expr, typ types.Type) {
	identifier, ok := target.(*ast.Ident)
	if !ok || identifier.Name == "_" || typ == nil {
		return
	}
	if object := b.unit.info.Defs[identifier]; object != nil {
		b.bindings[object] = typ
	}
}

func (b *loweringPlanBuilder) expressionType(expression ast.Expr) types.Type {
	if expression == nil {
		return nil
	}
	if typ := b.unit.info.TypeOf(expression); validPlannedType(typ) {
		return typ
	}
	return b.inferredExpressionType(expression)
}

func (b *loweringPlanBuilder) inferredExpressionType(expression ast.Expr) types.Type {
	switch node := expression.(type) {
	case *ast.ParenExpr:
		return b.expressionType(node.X)
	case *ast.Ident:
		return b.bindings[b.unit.info.ObjectOf(node)]
	case *ast.IndexExpr:
		return indexElementType(b.expressionType(node.X))
	case *ast.SelectorExpr:
		receiver := b.expressionType(node.X)
		if receiver != nil {
			object, _, _ := types.LookupFieldOrMethod(receiver, true, b.unit.typed, node.Sel.Name)
			if object != nil {
				return object.Type()
			}
		}
	case *ast.CallExpr:
		if signature := b.callSignature(node.Fun); signature != nil {
			if signature.Results().Len() == 1 {
				return signature.Results().At(0).Type()
			}
			return signature.Results()
		}
	case *ast.BinaryExpr:
		switch node.Op {
		case token.LAND, token.LOR, token.EQL, token.NEQ,
			token.LSS, token.LEQ, token.GTR, token.GEQ:
			return types.Typ[types.Bool]
		}
	}
	return nil
}

func indexElementType(container types.Type) types.Type {
	if container == nil {
		return nil
	}
	switch value := types.Unalias(container).Underlying().(type) {
	case *types.Array:
		return value.Elem()
	case *types.Slice:
		return value.Elem()
	case *types.Map:
		return value.Elem()
	case *types.Pointer:
		if array, ok := types.Unalias(value.Elem()).Underlying().(*types.Array); ok {
			return array.Elem()
		}
	}
	return nil
}

func validPlannedType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	if basic, ok := types.Unalias(typ).(*types.Basic); ok {
		return basic.Kind() != types.Invalid
	}
	return true
}

func (b *loweringPlanBuilder) callSignature(function ast.Expr) *types.Signature {
	signature, _ := types.Unalias(b.expressionType(function)).(*types.Signature)
	return signature
}

// selectEntry records operands that Go evaluates when it enters a select.
func (b *loweringPlanBuilder) selectEntry(statement ast.Stmt) *plannedBlock {
	if statement == nil {
		return nil
	}
	b.nextScope++
	block := &plannedBlock{scope: b.nextScope}
	operation := &plannedOperation{kind: planSourceStatement, source: statement}
	switch node := statement.(type) {
	case *ast.SendStmt:
		operation.expressions = b.expressionList([]ast.Expr{node.Chan, node.Value})
	case *ast.ExprStmt:
		if unary, ok := node.X.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
			operation.expressions = b.expressions(unary.X)
		}
	case *ast.AssignStmt:
		operation.expressions = b.expressionList(node.Rhs)
	}
	block.operations = append(block.operations, operation)
	return block
}

// selectSelected records receive targets that Go evaluates only after selection.
func (b *loweringPlanBuilder) selectSelected(statement ast.Stmt) *plannedBlock {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	b.nextScope++
	return &plannedBlock{
		scope: b.nextScope,
		operations: []*plannedOperation{{
			kind: planSourceStatement, source: statement,
			places: b.assignmentPlaces(assignment.Lhs...),
		}},
	}
}
