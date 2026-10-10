package compiler

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

func (e *loweringEmitter) emitPlannedAssignment(
	operation *plannedOperation,
	output *ast.BlockStmt,
) bool {
	assignment := operation.assignment
	if assignment == nil {
		return false
	}
	e.operations(assignment.prepare, output)
	if assignment.rhs != nil {
		e.operations(assignment.rhs, output)
	}
	if assignment.define {
		e.emitAssignmentDefinition(assignment, output)
		return true
	}
	for _, store := range assignment.stores {
		e.emitAssignmentStore(store, output)
	}
	return true
}

func (e *loweringEmitter) emitReceiveAssignment(
	communication *plannedCommunication,
	output *ast.BlockStmt,
) {
	assignment := communication.assignment
	if assignment == nil {
		return
	}
	e.operations(assignment.prepare, output)
	if assignment.define {
		right := make([]ast.Expr, 0, len(communication.receiveValues))
		for _, value := range communication.receiveValues {
			right = append(right, e.valueName(value.id, "received"))
		}
		e.emitDefinition(assignment.targets, right, output)
		return
	}
	for _, store := range assignment.stores {
		e.emitAssignmentStore(store, output)
	}
}

func (e *loweringEmitter) emitAssignmentDefinition(
	assignment *plannedAssignment,
	output *ast.BlockStmt,
) {
	right := make([]ast.Expr, 0, len(assignment.targets))
	for _, operand := range assignmentRightValues(assignment.right) {
		if operand.retained {
			value := e.expression(operand.expression, output)
			if operand.adapter {
				value = e.untypedBooleanAdapter(value)
			}
			right = append(right, value)
			continue
		}
		value := ast.Expr(e.valueName(operand.value.id, "result"))
		if operand.adapter {
			value = e.untypedBooleanAdapter(value)
		}
		right = append(right, value)
	}
	e.emitDefinition(assignment.targets, right, output)
}

func (e *loweringEmitter) emitDefinition(
	targets []plannedAssignmentTarget,
	right []ast.Expr,
	output *ast.BlockStmt,
) {
	left := make([]ast.Expr, 0, len(targets))
	for _, target := range targets {
		left = append(left, target.source)
	}
	output.List = append(output.List, &ast.AssignStmt{
		Lhs: left, Tok: token.DEFINE, Rhs: right,
	})
}

func (e *loweringEmitter) emitAssignmentStore(
	store plannedAssignmentStore,
	output *ast.BlockStmt,
) {
	e.emitPreparedPlaceOperation(store.place, output, func(
		place ast.Expr,
		block *ast.BlockStmt,
	) {
		if store.operator == token.INC || store.operator == token.DEC {
			block.List = append(block.List, &ast.IncDecStmt{X: place, Tok: store.operator})
			return
		}
		var value ast.Expr
		if store.retained {
			value = e.expression(store.expression, block)
		} else {
			value = e.valueName(store.value.id, "result")
		}
		if store.adapter {
			value = e.untypedBooleanAdapter(value)
		}
		block.List = append(block.List, &ast.AssignStmt{
			Lhs: []ast.Expr{place}, Tok: store.operator, Rhs: []ast.Expr{value},
		})
	})
}

func (e *loweringEmitter) emitAssignmentValueOperation(
	operation *plannedOperation,
	output *ast.BlockStmt,
) bool {
	switch operation.kind {
	case planPlaceReady:
		e.operations(operation.before, output)
	case planDeclareValue:
		value := e.plannedValue(operation.outputs[0])
		name := e.valueName(value.id, "place")
		output.List = append(output.List, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{name}, Type: e.contextTypeExpression(value, output),
			}},
		}})
	case planLoadPlace:
		e.emitPlaceLoad(operation, output)
	case planTypeKind:
		e.emitTypeKind(operation, output)
	default:
		return false
	}
	return true
}

func (e *loweringEmitter) emitPlaceLoad(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	value := e.plannedValue(operation.outputs[0])
	e.emitPreparedPlaceOperation(operation.places[0], output, func(
		place ast.Expr,
		block *ast.BlockStmt,
	) {
		if operation.declaresOutput {
			e.emitTypedExpressionBind(value, place, value.explicit, block)
			return
		}
		block.List = append(block.List, &ast.AssignStmt{
			Lhs: []ast.Expr{e.valueName(value.id, "place")},
			Tok: token.ASSIGN, Rhs: []ast.Expr{place},
		})
	})
}

func (e *loweringEmitter) emitTypeKind(
	operation *plannedOperation,
	output *ast.BlockStmt,
) {
	reference := operation.typeReference
	typeValue := plannedValue{
		typ: reference.typ, position: reference.use, typeReference: reference,
	}
	typeExpression := e.contextTypeExpression(typeValue, output)
	qualifier := e.packageQualifier(operation.packageRef)
	typeFor := &ast.IndexExpr{
		X:     &ast.SelectorExpr{X: ast.NewIdent(qualifier), Sel: ast.NewIdent("TypeFor")},
		Index: typeExpression,
	}
	kind := call(&ast.SelectorExpr{X: call(typeFor), Sel: ast.NewIdent("Kind")})
	isArray := &ast.BinaryExpr{
		X: kind, Op: token.EQL,
		Y: &ast.SelectorExpr{X: ast.NewIdent(qualifier), Sel: ast.NewIdent("Array")},
	}
	positionGeneratedExpression(isArray, reference.use)
	value := e.plannedValue(operation.outputs[0])
	e.emitTypedExpressionBind(value, isArray, false, output)
}

func (e *loweringEmitter) preparedPlaceExpression(place *plannedPlace) ast.Expr {
	switch place.kind {
	case planObjectPlace:
		return place.source
	case planDerefPlace:
		return &ast.StarExpr{X: e.valueName(place.values[0].id, "pointer")}
	case planFieldPlace:
		selector := place.source.(*ast.SelectorExpr)
		return &ast.SelectorExpr{
			X: e.preparedPlaceExpression(place.base), Sel: selector.Sel,
		}
	case planArrayIndexPlace:
		return &ast.IndexExpr{
			X:     e.preparedPlaceExpression(place.base),
			Index: e.valueName(place.values[0].id, "index"),
		}
	case planSliceIndexPlace, planMapIndexPlace:
		var index ast.Expr
		switch {
		case place.preparedIndex:
			index = e.valueName(place.values[1].id, "index")
		case place.retainIndex:
			index = place.index.source
		default:
			index = e.valueName(place.values[1].id, "index")
		}
		return &ast.IndexExpr{
			X:     e.valueName(place.values[0].id, "container"),
			Index: index,
		}
	}
	return nil
}

func (e *loweringEmitter) emitPreparedPlaceOperation(
	place *plannedPlace,
	output *ast.BlockStmt,
	emit func(ast.Expr, *ast.BlockStmt),
) {
	e.emitPreparedPlaceSuffix(place, output, func(value ast.Expr) ast.Expr { return value }, emit)
}

func (e *loweringEmitter) emitPreparedPlaceSuffix(
	place *plannedPlace,
	output *ast.BlockStmt,
	suffix func(ast.Expr) ast.Expr,
	emit func(ast.Expr, *ast.BlockStmt),
) {
	switch place.kind {
	case planFieldPlace:
		selector := place.source.(*ast.SelectorExpr)
		e.emitPreparedPlaceSuffix(place.base, output, func(base ast.Expr) ast.Expr {
			return suffix(&ast.SelectorExpr{X: base, Sel: selector.Sel})
		}, emit)
	case planArrayIndexPlace:
		index := e.valueName(place.values[0].id, "index")
		e.emitPreparedPlaceSuffix(place.base, output, func(base ast.Expr) ast.Expr {
			return suffix(&ast.IndexExpr{X: base, Index: index})
		}, emit)
	case planAlternativeIndexPlace:
		alternative := place.alternative
		index := e.valueName(alternative.index.id, "index")
		arrayBody := &ast.BlockStmt{}
		e.emitPreparedPlaceSuffix(alternative.original, arrayBody, func(base ast.Expr) ast.Expr {
			return suffix(&ast.IndexExpr{X: base, Index: index})
		}, emit)
		referenceBody := &ast.BlockStmt{}
		emit(suffix(&ast.IndexExpr{
			X: e.valueName(alternative.snapshot.id, "container"), Index: index,
		}), referenceBody)
		output.List = append(output.List, &ast.IfStmt{
			Cond: e.valueName(alternative.isArray.id, "isArray"), Body: arrayBody,
			Else: referenceBody,
		})
	default:
		emit(suffix(e.preparedPlaceExpression(place)), output)
	}
}

func (e *loweringEmitter) packageQualifier(reference plannedPackageReference) string {
	if name := e.packageQualifiers[reference.path]; name != "" {
		return name
	}
	names := make(map[string]bool, len(e.names))
	for name := range e.names {
		names[name] = true
	}
	ast.Inspect(e.source.File, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			names[identifier.Name] = true
		}
		return true
	})
	name := freshIdentifier(reference.preferred, names)
	e.names[name] = true
	e.packageQualifiers[reference.path] = name
	if reference.importSpec == nil {
		astutil.AddNamedImport(e.unit.fs, e.source.File, name, reference.path)
		return name
	}
	reference.importSpec.Name = ast.NewIdent(name)
	for _, identifier := range reference.identifiers {
		identifier.Name = name
	}
	return name
}
