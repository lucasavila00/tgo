package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/exp/typeparams"
)

type plannedAssignment struct {
	prepare     *plannedBlock
	rhs         *plannedBlock
	places      []*plannedPlace
	right       []plannedAssignmentRight
	stores      []plannedAssignmentStore
	targets     []plannedAssignmentTarget
	define      bool
	bindTargets bool
}

type plannedAssignmentRight struct {
	expression *plannedExpression
	values     []plannedValue
	expected   []plannedTypeReference
	retained   bool
	adapter    bool
	required   int
}

type plannedAssignmentStore struct {
	place      *plannedPlace
	value      plannedValue
	expression *plannedExpression
	retained   bool
	adapter    bool
	operator   token.Token
}

type plannedAssignmentOperand struct {
	value      plannedValue
	expression *plannedExpression
	retained   bool
	adapter    bool
}

type plannedAssignmentTarget struct {
	source ast.Expr
	object types.Object
	typ    types.Type
	define bool
	value  plannedValue
}

type plannedPlaceAlternative struct {
	original                 *plannedPlace
	natural                  types.Type
	isArray, snapshot, index plannedValue
}

func (b *loweringPlanBuilder) planDirectStatement(
	operation *plannedOperation,
	statement ast.Stmt,
) {
	if declaration, ok := statement.(*ast.DeclStmt); ok {
		operation.declarations = b.declarationPlans(declaration)
		for _, specification := range operation.declarations {
			operation.expressions = append(operation.expressions, specification.expressions...)
		}
	} else {
		operation.expressions = b.statementExpressions(statement)
		if _, assignment := statement.(*ast.AssignStmt); !assignment {
			if _, update := statement.(*ast.IncDecStmt); !update {
				operation.before = b.orderExpressions(operation.expressions)
			}
		}
	}
	if assignment, ok := statement.(*ast.AssignStmt); ok {
		operation.places = b.assignmentPlaces(assignment.Lhs...)
	}
	if update, ok := statement.(*ast.IncDecStmt); ok {
		operation.places = b.assignmentPlaces(update.X)
	}
	b.directStatementBinding(operation)
	b.planAssignment(operation)
}

func (b *loweringPlanBuilder) planAssignment(operation *plannedOperation) {
	switch source := operation.source.(type) {
	case *ast.AssignStmt:
		b.planAssignmentStatement(operation, source)
	case *ast.IncDecStmt:
		b.planUpdateStatement(operation, source)
	}
}

func (b *loweringPlanBuilder) planAssignmentStatement(
	operation *plannedOperation,
	source *ast.AssignStmt,
) {
	if !assignmentNeedsPlan(operation) {
		return
	}
	assignment := &plannedAssignment{
		prepare: &plannedBlock{scope: b.currentScope},
		rhs:     &plannedBlock{scope: b.currentScope}, places: operation.places,
	}
	expected := make([]types.Type, 0, len(source.Lhs))
	if source.Tok == token.DEFINE {
		assignment.define = true
		for _, target := range source.Lhs {
			planned := b.planAssignmentTarget(target)
			assignment.targets = append(assignment.targets, planned)
			expected = append(expected, planned.typ)
		}
	} else {
		for _, place := range assignment.places {
			expected = append(expected, place.typ)
		}
	}
	for _, place := range assignment.places {
		b.planPlacePreparation(place, assignment.prepare)
	}
	for index, expression := range operation.expressions {
		contexts := assignmentExpressionContexts(
			index, len(operation.expressions), expected,
		)
		assignment.right = append(assignment.right,
			b.planAssignmentRight(expression, contexts, assignment.rhs))
	}
	values := assignmentRightValues(assignment.right)
	if source.Tok == token.DEFINE {
		if len(values) != len(source.Lhs) {
			b.unit.failAt(source.Pos(), "assignment result count does not match targets")
			return
		}
		assignment.bindTargets = assignmentCanBindTargets(assignment, values)
	} else {
		if len(values) != len(operation.places) {
			b.unit.failAt(source.Pos(), "assignment result count does not match targets")
			return
		}
		for index, place := range operation.places {
			assignment.stores = append(assignment.stores, plannedAssignmentStore{
				place: place, value: values[index].value,
				expression: values[index].expression, retained: values[index].retained,
				adapter:  values[index].adapter,
				operator: source.Tok,
			})
		}
	}
	operation.assignment = assignment
	operation.before = nil
}

func (b *loweringPlanBuilder) planUpdateStatement(
	operation *plannedOperation,
	source *ast.IncDecStmt,
) {
	if !assignmentNeedsPlan(operation) {
		return
	}
	operation.assignment = &plannedAssignment{
		prepare: &plannedBlock{scope: b.currentScope}, places: operation.places,
		stores: []plannedAssignmentStore{{
			place: operation.places[0], operator: source.Tok,
		}},
	}
	b.planPlacePreparation(operation.places[0], operation.assignment.prepare)
	operation.before = nil
}

func assignmentCanBindTargets(
	assignment *plannedAssignment,
	values []plannedAssignmentOperand,
) bool {
	if len(assignment.targets) != len(values) {
		return false
	}
	for index := range assignment.targets {
		target := &assignment.targets[index]
		identifier, ok := target.source.(*ast.Ident)
		if !ok || identifier.Name == "_" || !target.define || values[index].retained {
			return false
		}
		target.value = values[index].value
	}
	return true
}

func assignmentNeedsPlan(operation *plannedOperation) bool {
	for _, expression := range operation.expressions {
		if plannedExpressionHasWork(expression) {
			return true
		}
	}
	for _, place := range operation.places {
		if plannedPlaceNeedsPreparation(place) {
			return true
		}
	}
	return false
}

func plannedPlaceNeedsPreparation(place *plannedPlace) bool {
	if place == nil {
		return false
	}
	if place.alternative != nil || plannedExpressionHasWork(place.container) ||
		plannedExpressionHasWork(place.index) {
		return true
	}
	return plannedPlaceNeedsPreparation(place.base)
}

func (b *loweringPlanBuilder) planReceiveAssignment(
	communication *plannedCommunication,
) *plannedAssignment {
	assignment := &plannedAssignment{
		prepare: &plannedBlock{scope: b.currentScope}, places: communication.targets,
	}
	if communication.token == token.DEFINE {
		assignment.define = true
		for _, target := range communication.left {
			assignment.targets = append(assignment.targets, b.planAssignmentTarget(target))
		}
		return assignment
	}
	if len(communication.receiveValues) != len(communication.targets) {
		b.unit.failAt(communication.source.Pos(),
			"receive result count does not match targets")
		return assignment
	}
	for index, place := range communication.targets {
		b.planPlacePreparation(place, assignment.prepare)
		assignment.stores = append(assignment.stores, plannedAssignmentStore{
			place: place, value: communication.receiveValues[index], operator: token.ASSIGN,
		})
	}
	return assignment
}

func (b *loweringPlanBuilder) planAssignmentRight(
	expression *plannedExpression,
	expected []types.Type,
	block *plannedBlock,
) plannedAssignmentRight {
	right := plannedAssignmentRight{
		expression: expression, required: len(expected),
	}
	b.planBooleanContextAdapter(expression)
	right.adapter = expression.booleanAdapter
	for _, typ := range expected {
		right.expected = append(right.expected,
			b.typeReference(typ, expression.source.Pos()))
	}
	if assignmentExpressionRetainsContext(b.unit.info, expression, expected) {
		right.retained = true
		return right
	}
	if expression.work != nil {
		right.values = append(right.values, expression.results...)
	} else {
		resultTypes := b.assignmentExpressionResultTypes(expression, expected)
		for index, typ := range resultTypes {
			value := b.newValue(typ, expression.source.Pos())
			value.explicit = expression.contextual && index < len(expected) &&
				expected[index] != nil
			b.plan.values[value.id-1] = value
			right.values = append(right.values, value)
		}
	}
	if len(right.values) != right.required {
		b.unit.failAt(expression.source.Pos(), "assignment result count does not match targets")
		return right
	}
	outputs := make([]valueID, 0, len(right.values))
	for _, value := range right.values {
		outputs = append(outputs, value.id)
	}
	block.operations = append(block.operations, &plannedOperation{
		kind: planEvaluate, expressions: []*plannedExpression{expression}, outputs: outputs,
		resultCount: right.required, contexts: right.expected,
	})
	return right
}

func assignmentExpressionContexts(
	index int,
	expressionCount int,
	expected []types.Type,
) []types.Type {
	if expressionCount == 1 {
		return append([]types.Type(nil), expected...)
	}
	if index >= len(expected) {
		return nil
	}
	return []types.Type{expected[index]}
}

func (b *loweringPlanBuilder) assignmentExpressionResultTypes(
	expression *plannedExpression,
	expected []types.Type,
) []types.Type {
	if assertion, ok := expression.source.(*ast.TypeAssertExpr); ok &&
		len(expected) == 2 {
		asserted := b.expressionType(assertion.Type)
		if identifier, ok := assertion.Type.(*ast.Ident); ok && asserted == nil {
			if object := b.unit.info.ObjectOf(identifier); object != nil {
				asserted = object.Type()
			}
		}
		if asserted == nil {
			asserted = expected[0]
		}
		return []types.Type{asserted, types.Typ[types.Bool]}
	}
	produced := plannedExpressionProducedType(expression)
	if tuple, ok := produced.(*types.Tuple); ok {
		result := make([]types.Type, 0, tuple.Len())
		for index := range tuple.Len() {
			result = append(result, tuple.At(index).Type())
		}
		return result
	}
	if len(expression.results) == len(expected) {
		result := make([]types.Type, 0, len(expression.results))
		for _, value := range expression.results {
			result = append(result, value.typ)
		}
		return result
	}
	if expression.contextual && len(expected) == 1 {
		return []types.Type{expected[0]}
	}
	return []types.Type{produced}
}

func assignmentExpressionRetainsContext(
	info *types.Info,
	expression *plannedExpression,
	expected []types.Type,
) bool {
	if expression == nil || expression.work != nil || expression.before != nil ||
		len(expected) != 1 {
		return false
	}
	value, ok := info.Types[expression.source]
	if ok && value.Value != nil {
		return true
	}
	identifier, ok := expression.source.(*ast.Ident)
	return ok && info.Uses[identifier] == types.Universe.Lookup("nil")
}

func assignmentRightValues(right []plannedAssignmentRight) []plannedAssignmentOperand {
	var values []plannedAssignmentOperand
	for _, item := range right {
		if item.retained {
			values = append(values, plannedAssignmentOperand{
				expression: item.expression, retained: true, adapter: item.adapter,
			})
			continue
		}
		for _, value := range item.values {
			values = append(values, plannedAssignmentOperand{value: value, adapter: item.adapter})
		}
	}
	return values
}

func (b *loweringPlanBuilder) planAssignmentTarget(expression ast.Expr) plannedAssignmentTarget {
	target := plannedAssignmentTarget{source: expression, typ: b.expressionType(expression)}
	if identifier, ok := expression.(*ast.Ident); ok {
		target.object = b.unit.info.Defs[identifier]
		target.define = target.object != nil
		if target.object == nil {
			target.object = b.unit.info.Uses[identifier]
		}
		if target.object != nil {
			target.typ = target.object.Type()
		}
	}
	return target
}

func (b *loweringPlanBuilder) planPlacePreparation(
	place *plannedPlace,
	block *plannedBlock,
) {
	if place == nil {
		return
	}
	if identifier, ok := place.source.(*ast.Ident); ok {
		place.object = b.unit.info.ObjectOf(identifier)
	}
	switch place.kind {
	case planObjectPlace:
		return
	case planDerefPlace:
		if place.base != nil {
			b.planPlacePreparation(place.base, block)
			b.planPlaceLoad(block, place.base, place.values[0], false)
			return
		}
		b.planAssignmentEvaluation(block, place.container, place.values[0])
	case planFieldPlace:
		b.planPlacePreparation(place.base, block)
	case planArrayIndexPlace:
		b.planPlacePreparation(place.base, block)
		b.planAssignmentEvaluation(block, place.index, place.values[0])
	case planSliceIndexPlace, planMapIndexPlace:
		b.planAssignmentEvaluation(block, place.container, place.values[0])
		if !place.retainIndex {
			b.planAssignmentEvaluation(block, place.index, place.values[1])
		}
	case planAlternativeIndexPlace:
		alternative := place.alternative
		b.planPlacePreparation(alternative.original, block)
		block.operations = append(block.operations, &plannedOperation{
			kind: planDeclareValue, outputs: []valueID{alternative.snapshot.id},
		})
		block.operations = append(block.operations, &plannedOperation{
			kind: planTypeKind, outputs: []valueID{alternative.isArray.id},
			typeReference: b.typeReference(alternative.natural, place.position),
			typeKind:      planArrayType, packageRef: b.packageReference("reflect", "reflect"),
		})
		snapshot := &plannedOperation{
			kind:    planLoadPlace,
			places:  []*plannedPlace{alternative.original},
			outputs: []valueID{alternative.snapshot.id},
		}
		block.operations = append(block.operations, &plannedOperation{
			kind: planBranch, inputs: []valueID{alternative.isArray.id},
			operator: token.LOR,
			body: &plannedBlock{
				scope: b.currentScope, operations: []*plannedOperation{snapshot},
			},
		})
		b.planAssignmentEvaluation(block, place.index, alternative.index)
	}
}

func (b *loweringPlanBuilder) planPlaceLoad(
	block *plannedBlock,
	place *plannedPlace,
	value plannedValue,
	declared bool,
) {
	if !declared && plannedPlaceContainsAlternative(place) {
		block.operations = append(block.operations, &plannedOperation{
			kind: planDeclareValue, outputs: []valueID{value.id},
		})
	}
	block.operations = append(block.operations, &plannedOperation{
		kind: planLoadPlace, places: []*plannedPlace{place}, outputs: []valueID{value.id},
		declaresOutput: !declared && !plannedPlaceContainsAlternative(place),
	})
}

func plannedPlaceContainsAlternative(place *plannedPlace) bool {
	if place == nil {
		return false
	}
	if place.kind == planAlternativeIndexPlace {
		return true
	}
	return plannedPlaceContainsAlternative(place.base)
}

func (b *loweringPlanBuilder) packageReference(
	path string,
	preferred string,
) plannedPackageReference {
	reference := plannedPackageReference{path: path, preferred: preferred}
	var object types.Object
	reference.importSpec, object = b.packageImport(path)
	if object == nil {
		if reference.importSpec != nil && importName(reference.importSpec) == "." {
			reference.identifiers = b.dotImportUses(path)
		}
		return reference
	}
	for identifier, used := range b.unit.info.Uses {
		if used == object {
			reference.identifiers = append(reference.identifiers, identifier)
		}
	}
	return reference
}

func (b *loweringPlanBuilder) packageImport(path string) (*ast.ImportSpec, types.Object) {
	for _, specification := range b.source.File.Imports {
		importPath, err := strconv.Unquote(specification.Path.Value)
		if err != nil || importPath != path {
			continue
		}
		if specification.Name == nil {
			return specification, b.unit.info.Implicits[specification]
		}
		return specification, b.unit.info.Defs[specification.Name]
	}
	return nil, nil
}

func importName(specification *ast.ImportSpec) string {
	if specification.Name == nil {
		return ""
	}
	return specification.Name.Name
}

func (b *loweringPlanBuilder) dotImportUses(path string) []*ast.Ident {
	var result []*ast.Ident
	for identifier, used := range b.unit.info.Uses {
		if used != nil && used.Pkg() != nil && used.Pkg().Path() == path &&
			used.Parent() == used.Pkg().Scope() {
			result = append(result, identifier)
		}
	}
	return result
}

func (b *loweringPlanBuilder) planAssignmentEvaluation(
	block *plannedBlock,
	expression *plannedExpression,
	value plannedValue,
) {
	block.operations = append(block.operations, &plannedOperation{
		kind: planEvaluate, expressions: []*plannedExpression{expression},
		outputs: []valueID{value.id},
	})
}

type indexStorageFamily uint8

const (
	indexStorageInvalid indexStorageFamily = iota
	indexStorageArray
	indexStorageReference
	indexStorageMap
)

func (b *loweringPlanBuilder) planTypeParameterIndexPlace(
	place *plannedPlace,
	typ types.Type,
) bool {
	parameter, ok := types.Unalias(typ).(*types.TypeParam)
	if !ok {
		return false
	}
	terms, err := typeparams.NormalTerms(parameter)
	if err != nil || len(terms) == 0 {
		return false
	}
	families, valid := indexStorageFamilies(terms)
	if !valid {
		return false
	}
	if families[indexStorageMap] {
		return len(families) == 1 && b.planTypeParameterMapPlace(place, typ, terms)
	}
	if len(families) == 1 && families[indexStorageArray] {
		place.kind = planArrayIndexPlace
		place.base = b.assignmentPlace(place.source.(*ast.IndexExpr).X)
		place.values = append(place.values, b.newValue(
			plannedExpressionProducedType(place.index), place.index.source.Pos(),
		))
		return true
	}
	if len(families) == 1 && families[indexStorageReference] {
		place.kind = planSliceIndexPlace
		place.values = append(place.values,
			b.newValue(typ, place.position),
			b.newValue(plannedExpressionProducedType(place.index), place.index.source.Pos()),
		)
		return true
	}
	if len(families) != 2 || !families[indexStorageArray] ||
		!families[indexStorageReference] {
		return false
	}
	place.kind = planAlternativeIndexPlace
	place.alternative = &plannedPlaceAlternative{
		original: b.assignmentPlace(place.source.(*ast.IndexExpr).X),
		natural:  typ,
		isArray:  b.newValue(types.Typ[types.Bool], place.position),
		snapshot: b.newValue(typ, place.position),
		index: b.newValue(
			plannedExpressionProducedType(place.index), place.index.source.Pos(),
		),
	}
	return true
}

func indexStorageFamilies(
	terms []*types.Term,
) (map[indexStorageFamily]bool, bool) {
	families := make(map[indexStorageFamily]bool)
	for _, term := range terms {
		family := indexTermStorageFamily(term.Type())
		if family == indexStorageInvalid {
			return nil, false
		}
		families[family] = true
	}
	return families, true
}

func (b *loweringPlanBuilder) planTypeParameterMapPlace(
	place *plannedPlace,
	typ types.Type,
	terms []*types.Term,
) bool {
	place.kind = planMapIndexPlace
	mapping := types.Unalias(terms[0].Type()).Underlying().(*types.Map)
	place.index = b.expressionContext(place.index.source, mapping.Key(), 1)
	place.retainIndex = assignmentExpressionRetainsContext(
		b.unit.info, place.index, []types.Type{mapping.Key()},
	)
	place.values = append(place.values, b.newValue(typ, place.position))
	if !place.retainIndex {
		place.values = append(place.values, b.newValue(
			plannedExpressionProducedType(place.index), place.index.source.Pos(),
		))
	}
	return true
}

func indexTermStorageFamily(typ types.Type) indexStorageFamily {
	switch item := types.Unalias(typ).Underlying().(type) {
	case *types.Array:
		return indexStorageArray
	case *types.Slice:
		return indexStorageReference
	case *types.Map:
		return indexStorageMap
	case *types.Pointer:
		if _, ok := item.Elem().Underlying().(*types.Array); ok {
			return indexStorageReference
		}
	}
	return indexStorageInvalid
}
