package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *loweringPlanBuilder) planExactComprehensionIndex(
	result *plannedExpression,
	assignment *ast.AssignStmt,
) {
	result.exact.index = b.newValue(types.Typ[types.Int], result.exact.outer.Pos())
	if outer, _ := plannedOperationForSource(result.work, result.exact.outer); outer != nil {
		outer.rangeKey = result.exact.index.id
	}
	operation, owner := plannedOperationForSource(result.work, assignment)
	if operation == nil {
		return
	}
	previousScope := b.currentScope
	b.currentScope = owner.scope
	defer func() { b.currentScope = previousScope }()

	container := b.expression(assignment.Lhs[0])
	elementType := underlyingSlice(result.typ).Elem()
	index := &plannedExpression{
		kind: planRetainedExpression, typ: types.Typ[types.Int], resultCount: 1,
		materialized: result.exact.index.id,
	}
	place := &plannedPlace{
		id: placeID(len(b.plan.places) + 1), kind: planSliceIndexPlace,
		typ: elementType, position: assignment.Pos(), container: container,
		index: index, preparedIndex: true,
	}
	place.values = append(place.values,
		b.newValue(plannedExpressionProducedType(container), assignment.Pos()),
		result.exact.index,
	)
	b.plan.places = append(b.plan.places, place)
	planned := &plannedAssignment{
		prepare: &plannedBlock{scope: owner.scope}, rhs: &plannedBlock{scope: owner.scope},
		places: []*plannedPlace{place},
	}
	b.planPlacePreparation(place, planned.prepare)
	element := b.expressionContext(result.exact.appendCall.Args[1], elementType, 1)
	planned.right = append(planned.right,
		b.planAssignmentRight(element, []types.Type{elementType}, planned.rhs))
	value := assignmentRightValues(planned.right)[0]
	planned.stores = append(planned.stores, plannedAssignmentStore{
		place: place, value: value.value, expression: value.expression,
		retained: value.retained, adapter: value.adapter, operator: token.ASSIGN,
	})
	operation.expressions = []*plannedExpression{element}
	operation.assignment = planned
	operation.before = nil
}

func plannedOperationForSource(
	block *plannedBlock,
	source ast.Stmt,
) (*plannedOperation, *plannedBlock) {
	if block == nil {
		return nil, nil
	}
	for _, operation := range block.operations {
		if operation.source == source {
			return operation, block
		}
		for _, child := range []*plannedBlock{
			operation.init, operation.test, operation.body, operation.post,
			operation.otherwise, operation.before, operation.after,
		} {
			if found, owner := plannedOperationForSource(child, source); found != nil {
				return found, owner
			}
		}
		childrenGroups := [][]*plannedBlock{
			operation.cases, operation.entry, operation.selected,
		}
		for _, children := range childrenGroups {
			for _, child := range children {
				if found, owner := plannedOperationForSource(child, source); found != nil {
					return found, owner
				}
			}
		}
	}
	return nil, nil
}
