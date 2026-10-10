package tgolint

import (
	"go/token"
	"go/types"
	"testing"
)

func TestEnumFieldRegionsKeepOwnersSeparate(t *testing.T) {
	t.Parallel()
	table := newEnumIdentityTable()
	first := table.region(enumRegionKey{site: 1})
	second := table.region(enumRegionKey{site: 2})
	field := types.NewVar(token.NoPos, nil, "Value", types.Typ[types.Int])
	left := table.fieldRegion(first, field, 0)
	right := table.fieldRegion(second, field, 0)
	if left == right {
		t.Fatal("field regions for separate owners are equal")
	}
}

func TestEnumProofKeepsTestedTargetRelation(t *testing.T) {
	t.Parallel()
	state := newEnumEventState()
	selected := enumAbstractValue{
		regions:       enumRegionSet{1: true, 2: true},
		dependencies:  enumCellSet{3: true},
		relations:     enumCellSet{3: true},
		relationKnown: true,
		readCells:     enumCellSet{3: true},
	}
	selected.observation = state.observe(selected)
	state.prove(selected, 1)
	if !state.payloadValid(selected, 1) {
		t.Fatal("the tested receiver lost its proof")
	}
	if state.payloadValid(enumAbstractValue{regions: enumRegionSet{1: true}}, 1) {
		t.Fatal("a possible target inherited the receiver proof")
	}
	independent := enumAbstractValue{
		regions:       enumRegionSet{1: true, 2: true},
		dependencies:  enumCellSet{4: true},
		relations:     enumCellSet{4: true},
		relationKnown: true,
		readCells:     enumCellSet{4: true},
	}
	if state.payloadValid(independent, 1) {
		t.Fatal("an independent selection inherited the receiver proof")
	}
	alias := cloneEnumAbstractValue(selected)
	alias.dependencies[5] = true
	if !state.payloadValid(alias, 1) {
		t.Fatal("a copied alias lost the receiver proof")
	}
}

func TestEnumWriteStalesStoredTagObservation(t *testing.T) {
	t.Parallel()
	state := newEnumEventState()
	value := enumAbstractValue{regions: enumRegionSet{1: true}}
	value.observation = state.observe(value)
	state.cells[1] = cloneEnumAbstractValue(value)
	state.invalidateObservations(nil, enumRegionSet{1: true})
	if !state.cells[1].observation.stale {
		t.Fatal("the stored tag observation stayed fresh")
	}
}

func TestEnumPointerRebindKeepsPointeeProof(t *testing.T) {
	t.Parallel()
	graph := newEnumEventGraph()
	state := newEnumEventState()
	proofValue := enumAbstractValue{
		regions:       enumRegionSet{1: true},
		dependencies:  enumCellSet{2: true},
		relations:     enumCellSet{2: true},
		relationKnown: true,
		readCells:     enumCellSet{2: true},
	}
	proofValue.observation = state.observe(proofValue)
	state.prove(proofValue, 1)
	state.cells[1] = enumAbstractValue{regions: enumRegionSet{1: true}}
	state.saved[1] = enumAbstractValue{regions: enumRegionSet{2: true}}
	graph.transfer(state, enumEvent{
		kind:  enumEventStore,
		cells: enumCellSet{1: true},
		value: 1,
	})
	if !state.payloadValid(proofValue, 1) {
		t.Fatal("a pointer rebind removed the pointee proof")
	}
}

func TestEnumClosureInputIncludesCapturedState(t *testing.T) {
	t.Parallel()
	graph := newEnumEventGraph()
	worklist := newEnumEventCallWorklist(graph)
	call := worklist.call(enumEventCallKey{closure: 1, caller: 1})
	first := newEnumEventState()
	first.cells[1] = enumAbstractValue{regions: enumRegionSet{1: true}}
	worklist.addInput(call, nil, first)
	second := newEnumEventState()
	second.cells[1] = enumAbstractValue{regions: enumRegionSet{2: true}}
	worklist.addInput(call, nil, second)
	captured := worklist.summary(call).captures[1].regions
	if !captured[1] || !captured[2] {
		t.Fatalf("captured regions = %v", captured)
	}
}

func TestEnumRelationRequiresEveryJoinedPath(t *testing.T) {
	t.Parallel()
	left := newEnumEventState()
	left.cells[3] = enumAbstractValue{
		regions:       enumRegionSet{1: true, 2: true},
		relations:     enumCellSet{1: true},
		relationKnown: true,
	}
	right := newEnumEventState()
	right.cells[3] = enumAbstractValue{
		regions:       enumRegionSet{1: true},
		relations:     enumCellSet{2: true},
		relationKnown: true,
	}
	joined, _ := joinEnumEventStates(left, right)
	if !enumCellSetEqual(joined.cells[3].relations, enumCellSet{3: true}) {
		t.Fatalf("joined relation = %v", joined.cells[3].relations)
	}
	if _, changed := joinEnumEventStates(joined, right); changed {
		t.Fatal("the finite phi relation did not converge")
	}
	unchanged, _ := joinEnumEventStates(left, left)
	if !enumCellSetEqual(unchanged.cells[3].relations, enumCellSet{1: true}) {
		t.Fatalf("unchanged relation = %v", unchanged.cells[3].relations)
	}
	proofState := newEnumEventState()
	tested := enumAbstractValue{
		regions:       enumRegionSet{1: true, 2: true},
		relations:     enumCellSet{1: true},
		relationKnown: true,
		readCells:     enumCellSet{1: true},
	}
	tested.observation = proofState.observe(tested)
	proofState.prove(tested, 1)
	partial := joined.cells[3]
	if proofState.payloadValid(partial, 1) {
		t.Fatal("a partially rebound copy inherited the tested proof")
	}
}

func TestEnumCopiedValueIgnoresSourceRebind(t *testing.T) {
	t.Parallel()
	graph := newEnumEventGraph()
	state := newEnumEventState()
	copy := enumAbstractValue{
		regions:       enumRegionSet{1: true},
		relations:     enumCellSet{1: true},
		relationKnown: true,
		readCells:     enumCellSet{2: true},
	}
	copy.observation = state.observe(copy)
	state.prove(copy, 1)
	state.saved[1] = enumAbstractValue{
		regions:       enumRegionSet{2: true},
		relations:     enumCellSet{3: true},
		relationKnown: true,
	}
	graph.transfer(state, enumEvent{
		kind:  enumEventStore,
		cells: enumCellSet{1: true},
		value: 1,
	})
	if !state.payloadValid(copy, 1) {
		t.Fatal("rebinding the source removed the copied value proof")
	}
}

func TestEnumInvocationStartsWithNoPriorWrites(t *testing.T) {
	t.Parallel()
	input := newEnumEventState()
	input.cellWrites[1] = enumWriteSet{1: true}
	input.regionWrites[1] = enumWriteSet{1: true}
	invocation := enumInvocationState(input)
	if len(invocation.cellWrites) != 0 || len(invocation.regionWrites) != 0 {
		t.Fatal("the invocation retained writes from before the call")
	}
}

func TestEnumRecursiveFieldUsesOneSummaryRegion(t *testing.T) {
	t.Parallel()
	table := newEnumIdentityTable()
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "Node", nil), nil, nil,
	)
	pointer := types.NewPointer(named)
	field := types.NewVar(token.NoPos, nil, "Next", pointer)
	named.SetUnderlying(types.NewStruct([]*types.Var{field}, nil))
	owner := table.region(enumRegionKey{site: 1, typ: pointer})
	first := table.fieldRegion(owner, field, 1)
	second := table.fieldRegion(first, field, 1)
	if first != second || table.uniqueRegion(first) {
		t.Fatalf("recursive regions = %d, %d", first, second)
	}
}
