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
	selected := enumAbstractValue{regions: enumRegionSet{1: true, 2: true}}
	selected.observation = state.observe(selected)
	state.prove(selected, 1)
	if !state.payloadValid(selected, 1) {
		t.Fatal("the tested receiver lost its proof")
	}
	if state.payloadValid(enumAbstractValue{regions: enumRegionSet{1: true}}, 1) {
		t.Fatal("a possible target inherited the receiver proof")
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
		regions:      enumRegionSet{1: true},
		dependencies: enumCellSet{2: true},
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
	worklist.addInput(call, nil, map[enumCellID]enumAbstractValue{
		1: {regions: enumRegionSet{1: true}},
	})
	worklist.addInput(call, nil, map[enumCellID]enumAbstractValue{
		1: {regions: enumRegionSet{2: true}},
	})
	captured := worklist.summary(call).captures[1].regions
	if !captured[1] || !captured[2] {
		t.Fatalf("captured regions = %v", captured)
	}
}
