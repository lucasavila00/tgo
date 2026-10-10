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
	place := b.newAssignmentPlace(expression)
	b.plan.places = append(b.plan.places, place)
	switch node := expression.(type) {
	case *ast.StarExpr:
		place.kind = planDerefPlace
		place.container = b.expression(node.X)
		place.values = append(place.values, b.newValue(
			plannedExpressionProducedType(place.container), node.X.Pos(),
		))
	case *ast.SelectorExpr:
		place.kind = planFieldPlace
		if selection := b.unit.info.Selections[node]; selection != nil &&
			len(selection.Index()) > 1 {
			place.base = b.promotedSelectionBase(node, selection)
		} else if underlyingPointer(b.expressionType(node.X)) != nil {
			place.base = b.derefPlace(node.X)
		} else {
			place.base = b.assignmentPlace(node.X)
		}
	case *ast.IndexExpr:
		containerPlan := b.expression(node.X)
		producedType := plannedExpressionProducedType(containerPlan)
		place.container = containerPlan
		place.index = b.expression(node.Index)
		if b.planTypeParameterIndexPlace(place, producedType) {
			return place
		}
		underlyingType := coreContainerType(producedType)
		switch containerType := underlyingType.(type) {
		case *types.Array:
			place.kind = planArrayIndexPlace
			place.base = b.assignmentPlace(node.X)
		case *types.Pointer:
			place.kind = planArrayIndexPlace
			place.base = b.derefPlace(node.X)
		case *types.Slice:
			place.kind = planSliceIndexPlace
			place.container = containerPlan
			place.values = append(place.values, b.newValue(producedType, node.X.Pos()))
		case *types.Map:
			place.kind = planMapIndexPlace
			place.container = containerPlan
			place.index = b.expressionContext(node.Index, containerType.Key(), 1)
			place.retainIndex = assignmentExpressionRetainsContext(
				b.unit.info, place.index, []types.Type{containerType.Key()},
			)
			place.values = append(place.values, b.newValue(producedType, node.X.Pos()))
		}
		if !place.retainIndex {
			place.values = append(place.values, b.newValue(
				plannedExpressionProducedType(place.index), node.Index.Pos(),
			))
		}
	}
	return place
}

func (b *loweringPlanBuilder) newAssignmentPlace(expression ast.Expr) *plannedPlace {
	place := &plannedPlace{
		id: placeID(len(b.plan.places) + 1), typ: b.expressionType(expression),
		position: expression.Pos(), source: expression, kind: planObjectPlace,
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return place
	}
	place.object = b.unit.info.ObjectOf(identifier)
	if place.object != nil {
		place.typ = place.object.Type()
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
	if underlyingPointer(b.expressionType(node.X)) != nil {
		base = b.derefPlace(node.X)
		expressionPlan = &plannedExpression{
			kind: planRetainedExpression, source: node.X,
			typ: b.expressionType(node.X), materialized: base.values[0].id, resultCount: 1,
		}
	}
	for _, fieldIndex := range selection.Index()[:len(selection.Index())-1] {
		if pointer := underlyingPointer(typ); pointer != nil {
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
		if underlyingPointer(field.Type()) != nil {
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
	pointer := underlyingPointer(typ)
	if pointer == nil {
		return nil
	}
	return pointer.Elem()
}

func underlyingPointer(typ types.Type) *types.Pointer {
	pointer, _ := coreContainerType(typ).(*types.Pointer)
	return pointer
}

func coreContainerType(typ types.Type) types.Type {
	typ = types.Unalias(typ)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	switch item := typ.(type) {
	case *types.Array, *types.Slice, *types.Map, *types.Pointer:
		return item
	case *types.TypeParam:
		return coreContainerType(item.Constraint())
	case *types.Interface:
		var core types.Type
		for index := range item.NumEmbeddeds() {
			next := coreContainerType(item.EmbeddedType(index))
			if next == nil {
				continue
			}
			if core != nil && !types.Identical(core, next) {
				return nil
			}
			core = next
		}
		return core
	case *types.Union:
		var core types.Type
		for index := range item.Len() {
			next := coreContainerType(item.Term(index).Type())
			if next == nil || core != nil && !types.Identical(core, next) {
				return nil
			}
			core = next
		}
		return core
	}
	return nil
}

func (b *loweringPlanBuilder) derefPlace(expression ast.Expr) *plannedPlace {
	container := b.expression(expression)
	pointerType := plannedExpressionProducedType(container)
	typ := types.Type(nil)
	if pointer := underlyingPointer(pointerType); pointer != nil {
		typ = pointer.Elem()
	}
	place := &plannedPlace{
		id: placeID(len(b.plan.places) + 1), typ: typ, position: expression.Pos(),
		kind: planDerefPlace, source: expression, container: container,
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
