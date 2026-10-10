package tgolint

import (
	"sort"
	"strconv"
	"strings"
	"go/types"

	"tgo/pkg/syntax"
)

type enumRegionProof struct {
	targets  string
	relation string
	tag      int
}

type enumEventState struct {
	cells             map[enumCellID]enumAbstractValue
	saved             map[enumSavedValueID]enumAbstractValue
	proofs            map[enumRegionProof]bool
	proofTargets      map[enumRegionProof]enumRegionSet
	proofDependencies map[enumRegionProof]enumCellSet
	regionWrites      map[enumRegionID]enumWriteSet
	cellWrites        map[enumCellID]enumWriteSet
}

func cloneEnumCellSet(source enumCellSet) enumCellSet {
	result := make(enumCellSet, len(source))
	for cell := range source {
		result[cell] = true
	}
	return result
}

func cloneEnumRegionSet(source enumRegionSet) enumRegionSet {
	result := make(enumRegionSet, len(source))
	for region := range source {
		result[region] = true
	}
	return result
}

func equalEnumWriteSet(left, right enumWriteSet) bool {
	if len(left) != len(right) {
		return false
	}
	for write := range left {
		if !right[write] {
			return false
		}
	}
	return true
}

func equalEnumTagObservation(
	left *enumTagObservation,
	right *enumTagObservation,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if left.stale != right.stale {
		return false
	}
	if len(left.regions) != len(right.regions) || len(left.cells) != len(right.cells) {
		return false
	}
	for region, writes := range left.regions {
		if !equalEnumWriteSet(writes, right.regions[region]) {
			return false
		}
	}
	for cell, writes := range left.cells {
		if !equalEnumWriteSet(writes, right.cells[cell]) {
			return false
		}
	}
	return true
}

func newEnumEventState() *enumEventState {
	return &enumEventState{
		cells:             make(map[enumCellID]enumAbstractValue),
		saved:             make(map[enumSavedValueID]enumAbstractValue),
		proofs:            make(map[enumRegionProof]bool),
		proofTargets:      make(map[enumRegionProof]enumRegionSet),
		proofDependencies: make(map[enumRegionProof]enumCellSet),
		regionWrites:      make(map[enumRegionID]enumWriteSet),
		cellWrites:        make(map[enumCellID]enumWriteSet),
	}
}

func cloneEnumEventState(source *enumEventState) *enumEventState {
	if source == nil {
		return nil
	}
	result := newEnumEventState()
	for cell, value := range source.cells {
		result.cells[cell] = cloneEnumAbstractValue(value)
	}
	for saved, value := range source.saved {
		result.saved[saved] = cloneEnumAbstractValue(value)
	}
	for proof := range source.proofs {
		result.proofs[proof] = true
		result.proofTargets[proof] = cloneEnumRegionSet(source.proofTargets[proof])
		result.proofDependencies[proof] = cloneEnumCellSet(
			source.proofDependencies[proof],
		)
	}
	for region, writes := range source.regionWrites {
		result.regionWrites[region] = cloneEnumWriteSet(writes)
	}
	for cell, writes := range source.cellWrites {
		result.cellWrites[cell] = cloneEnumWriteSet(writes)
	}
	return result
}

func joinEnumAbstractValues(
	left enumAbstractValue,
	right enumAbstractValue,
) (enumAbstractValue, bool) {
	result := cloneEnumAbstractValue(left)
	changed := false
	if result.regions == nil {
		result.regions = make(enumRegionSet)
	}
	for region := range right.regions {
		if !result.regions[region] {
			result.regions[region] = true
			changed = true
		}
	}
	if result.places == nil {
		result.places = make(enumCellSet)
	}
	for cell := range right.places {
		if !result.places[cell] {
			result.places[cell] = true
			changed = true
		}
	}
	if result.readCells == nil {
		result.readCells = make(enumCellSet)
	}
	for cell := range right.readCells {
		if !result.readCells[cell] {
			result.readCells[cell] = true
			changed = true
		}
	}
	if !result.relationKnown && right.relationKnown {
		result.relations = cloneEnumCellSet(right.relations)
		result.relationKnown = true
		changed = true
	} else if result.relationKnown && right.relationKnown {
		intersection := intersectEnumCellSets(result.relations, right.relations)
		if !enumCellSetEqual(intersection, result.relations) {
			result.relations = intersection
			changed = true
		}
	}
	if result.closures == nil {
		result.closures = make(enumEventClosureSet)
	}
	for closure := range right.closures {
		if !result.closures[closure] {
			result.closures[closure] = true
			changed = true
		}
	}
	if result.dependencies == nil {
		result.dependencies = make(enumCellSet)
	}
	for cell := range right.dependencies {
		if !result.dependencies[cell] {
			result.dependencies[cell] = true
			changed = true
		}
	}
	var observationChanged bool
	result.observation, observationChanged = joinEnumTagObservations(
		result.observation, right.observation,
	)
	changed = changed || observationChanged
	if right.unknown && !result.unknown {
		result.unknown = true
		changed = true
	}
	return result, changed
}

func joinEnumTagObservations(
	left *enumTagObservation,
	right *enumTagObservation,
) (*enumTagObservation, bool) {
	if left == nil || right == nil {
		return nil, left != nil
	}
	result := cloneEnumTagObservation(left)
	changed := false
	if right.stale && !result.stale {
		result.stale = true
		changed = true
	}
	for region, writes := range right.regions {
		if _, found := result.regions[region]; !found {
			result.regions[region] = cloneEnumWriteSet(writes)
			changed = true
		}
	}
	for cell, writes := range right.cells {
		if _, found := result.cells[cell]; !found {
			result.cells[cell] = cloneEnumWriteSet(writes)
			changed = true
		}
	}
	return result, changed
}

// joinEnumEventStates keeps only proofs that hold on every incoming path.
func joinEnumEventStates(
	current *enumEventState,
	incoming *enumEventState,
) (*enumEventState, bool) {
	if current == nil {
		return cloneEnumEventState(incoming), true
	}
	result := cloneEnumEventState(current)
	changed := false
	for cell, value := range incoming.cells {
		currentValue, present := result.cells[cell]
		joined, added := joinEnumAbstractValues(currentValue, value)
		if present && currentValue.relationKnown && value.relationKnown &&
			!enumCellSetEqual(currentValue.relations, value.relations) {
			joined.relations = enumCellSet{cell: true}
			joined.relationKnown = true
			added = true
		}
		result.cells[cell] = joined
		changed = changed || added
	}
	for saved, value := range incoming.saved {
		joined, added := joinEnumAbstractValues(result.saved[saved], value)
		result.saved[saved] = joined
		changed = changed || added
	}
	for proof := range result.proofs {
		if !incoming.proofs[proof] {
			delete(result.proofs, proof)
			delete(result.proofTargets, proof)
			delete(result.proofDependencies, proof)
			changed = true
		}
	}
	for proof := range result.proofs {
		for cell := range incoming.proofDependencies[proof] {
			if result.proofDependencies[proof] == nil {
				result.proofDependencies[proof] = make(enumCellSet)
			}
			if !result.proofDependencies[proof][cell] {
				result.proofDependencies[proof][cell] = true
				changed = true
			}
		}
	}
	for region, writes := range incoming.regionWrites {
		joined, added := joinEnumWriteSets(result.regionWrites[region], writes)
		result.regionWrites[region] = joined
		changed = changed || added
	}
	for cell, writes := range incoming.cellWrites {
		joined, added := joinEnumWriteSets(result.cellWrites[cell], writes)
		result.cellWrites[cell] = joined
		changed = changed || added
	}
	return result, changed
}

func joinEnumWriteSets(left, right enumWriteSet) (enumWriteSet, bool) {
	result := cloneEnumWriteSet(left)
	changed := false
	for write := range right {
		if !result[write] {
			result[write] = true
			changed = true
		}
	}
	return result, changed
}

func intersectEnumCellSets(left, right enumCellSet) enumCellSet {
	result := make(enumCellSet)
	for cell := range left {
		if right[cell] {
			result[cell] = true
		}
	}
	return result
}

func enumCellSetEqual(left, right enumCellSet) bool {
	if len(left) != len(right) {
		return false
	}
	for cell := range left {
		if !right[cell] {
			return false
		}
	}
	return true
}

func (state *enumEventState) killRegionProofs(regions enumRegionSet) {
	for proof := range state.proofs {
		for region := range state.proofTargets[proof] {
			if regions[region] {
				delete(state.proofs, proof)
				delete(state.proofTargets, proof)
				delete(state.proofDependencies, proof)
				break
			}
		}
	}
}

func (state *enumEventState) prove(value enumAbstractValue, tag int) {
	if value.unknown || !state.observationFresh(value.observation) {
		return
	}
	if !value.relationKnown || len(value.relations) == 0 {
		return
	}
	proof := enumRegionProof{
		targets:  enumRegionSignature(value.regions),
		relation: enumCellSignature(value.relations),
		tag:      tag,
	}
	state.proofs[proof] = true
	state.proofTargets[proof] = cloneEnumRegionSet(value.regions)
	state.proofDependencies[proof] = cloneEnumCellSet(value.readCells)
}

func (state *enumEventState) observationFresh(
	observation *enumTagObservation,
) bool {
	if observation == nil || observation.stale {
		return false
	}
	return true
}

func (state *enumEventState) payloadValid(
	value enumAbstractValue,
	tag int,
) bool {
	if value.unknown || len(value.regions) == 0 {
		return false
	}
	targets := enumRegionSignature(value.regions)
	relation := enumCellSignature(value.relations)
	for proof := range state.proofs {
		if value.relationKnown && proof.targets == targets &&
			proof.relation == relation && proof.tag == tag {
			return true
		}
	}
	return false
}

func enumRegionSignature(regions enumRegionSet) string {
	values := make([]int, 0, len(regions))
	for region := range regions {
		values = append(values, int(region))
	}
	sort.Ints(values)
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func enumCellSignature(cells enumCellSet) string {
	values := make([]int, 0, len(cells))
	for cell := range cells {
		values = append(values, int(cell))
	}
	sort.Ints(values)
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func enumCellSubset(left, right enumCellSet) bool {
	for cell := range left {
		if !right[cell] {
			return false
		}
	}
	return true
}

func (graph *enumEventGraph) transfer(
	state *enumEventState,
	event enumEvent,
) bool {
	switch event.kind {
	case enumEventResolvePlace:
		base := state.saved[event.source]
		value := enumAbstractValue{places: make(enumCellSet)}
		for region := range base.regions {
			cell := graph.identities.cell(enumCellKey{
				owner: region,
				field: event.field,
			})
			if _, found := state.cells[cell]; !found {
				if _, pointer := types.Unalias(event.field.Type()).(*types.Pointer); pointer {
					target := graph.identities.fieldRegion(
						region, event.field, graph.activation,
					)
					state.cells[cell] = enumAbstractValue{
						regions:       enumRegionSet{target: true},
						dependencies:  enumCellSet{cell: true},
						relations:     enumCellSet{cell: true},
						relationKnown: true,
					}
				}
			}
			value.places[cell] = true
		}
		state.saved[event.value] = value
	case enumEventLoad:
		value := enumAbstractValue{}
		readCells := cloneEnumCellSet(event.cells)
		for cell := range event.cells {
			value, _ = joinEnumAbstractValues(value, state.cells[cell])
			if value.dependencies == nil {
				value.dependencies = make(enumCellSet)
			}
			value.dependencies[cell] = true
		}
		for cell := range state.saved[event.source].places {
			readCells[cell] = true
			value, _ = joinEnumAbstractValues(value, state.cells[cell])
			if value.dependencies == nil {
				value.dependencies = make(enumCellSet)
			}
			value.dependencies[cell] = true
		}
		value.readCells = readCells
		state.saved[event.value] = value
	case enumEventTagRead:
		value := cloneEnumAbstractValue(state.saved[event.source])
		value.observation = state.observe(value)
		state.saved[event.value] = value
	case enumEventSave:
		state.saved[event.value] = graph.saved(event.value)
	case enumEventCall:
		arguments := make([]enumAbstractValue, len(event.arguments))
		for index, saved := range event.arguments {
			arguments[index] = cloneEnumAbstractValue(state.saved[saved])
		}
		callee := state.saved[event.value]
		if len(callee.closures) != 0 {
			graph.calls.applyAlternatives(
				state, callee.closures, arguments, event.results,
			)
			break
		}
		regions := make(enumRegionSet)
		if event.method {
			for region := range callee.regions {
				regions[region] = true
			}
		}
		for _, argument := range arguments {
			if len(argument.closures) != 0 {
				graph.calls.applyAlternatives(
					state, argument.closures, nil, nil,
				)
			}
			for region := range argument.regions {
				regions[region] = true
			}
		}
		state.killRegionProofs(regions)
		state.invalidateObservations(nil, regions)
		for region := range regions {
			state.recordRegionWrite(region, event.serial)
		}
		for _, result := range event.results {
			state.saved[result] = enumAbstractValue{unknown: true}
		}
	case enumEventStore:
		value := state.saved[event.value]
		cells := cloneEnumCellSet(event.cells)
		regions := cloneEnumRegionSet(event.regions)
		if len(event.values) != 0 {
			destination := state.saved[event.values[0]]
			for cell := range destination.places {
				cells[cell] = true
			}
			for region := range destination.regions {
				regions[region] = true
			}
		}
		strong := len(cells) == 1
		for cell := range cells {
			old := state.cells[cell]
			state.recordCellWrite(cell, event.serial)
			if strong && graph.uniqueCell(cell) {
				state.cells[cell] = cloneEnumAbstractValue(value)
			} else {
				state.cells[cell], _ = joinEnumAbstractValues(old, value)
			}
		}
		state.invalidateObservations(cells, regions)
		for region := range regions {
			state.killRegionProofs(enumRegionSet{region: true})
			state.recordRegionWrite(region, event.serial)
		}
	case enumEventTagTest:
		state.prove(state.saved[event.value], event.tag)
	case enumEventPayloadCheck:
		return state.payloadValid(state.saved[event.value], event.tag)
	}
	return true
}

func (state *enumEventState) invalidateObservations(
	cells enumCellSet,
	regions enumRegionSet,
) {
	for id, value := range state.saved {
		state.saved[id] = staleEnumObservation(value, cells, regions)
	}
	for cell, value := range state.cells {
		state.cells[cell] = staleEnumObservation(value, cells, regions)
	}
}

func staleEnumObservation(
	value enumAbstractValue,
	cells enumCellSet,
	regions enumRegionSet,
) enumAbstractValue {
	observation := value.observation
	if observation == nil || observation.stale {
		return value
	}
	invalid := false
	for cell := range cells {
		if _, depends := observation.cells[cell]; depends {
			invalid = true
			break
		}
	}
	if !invalid {
		for region := range regions {
			if _, depends := observation.regions[region]; depends {
				invalid = true
				break
			}
		}
	}
	if invalid {
		value.observation = cloneEnumTagObservation(observation)
		value.observation.stale = true
	}
	return value
}

func (state *enumEventState) observe(value enumAbstractValue) *enumTagObservation {
	result := &enumTagObservation{
		regions: make(map[enumRegionID]enumWriteSet, len(value.regions)),
		cells:   make(map[enumCellID]enumWriteSet, len(value.readCells)),
	}
	for region := range value.regions {
		result.regions[region] = cloneEnumWriteSet(state.regionWrites[region])
	}
	for cell := range value.readCells {
		result.cells[cell] = cloneEnumWriteSet(state.cellWrites[cell])
	}
	return result
}

func (state *enumEventState) recordCellWrite(cell enumCellID, serial int) {
	if state.cellWrites[cell] == nil {
		state.cellWrites[cell] = make(enumWriteSet)
	}
	state.cellWrites[cell][serial] = true
	for proof, dependencies := range state.proofDependencies {
		if dependencies[cell] {
			delete(state.proofs, proof)
			delete(state.proofTargets, proof)
			delete(state.proofDependencies, proof)
		}
	}
}

func (state *enumEventState) recordRegionWrite(region enumRegionID, serial int) {
	if state.regionWrites[region] == nil {
		state.regionWrites[region] = make(enumWriteSet)
	}
	state.regionWrites[region][serial] = true
}

func (graph *enumEventGraph) uniqueCell(cell enumCellID) bool {
	index := int(cell) - 1
	if index < 0 || index >= len(graph.identities.cellKeys) {
		return false
	}
	key := graph.identities.cellKeys[index]
	return key.owner == 0 || graph.identities.uniqueRegion(key.owner)
}

type enumEventResult struct {
	access map[*syntax.Expression]bool
	before map[enumEventLocation]*enumEventState
	output *enumEventState
	values []enumAbstractValue
}

// run evaluates all event paths to a finite fixed point.
func (graph *enumEventGraph) run(initial *enumEventState) *enumEventResult {
	result := &enumEventResult{
		access: make(map[*syntax.Expression]bool),
		before: make(map[enumEventLocation]*enumEventState),
	}
	if initial == nil {
		initial = graph.initial
	}
	if graph.calls == nil {
		graph.calls = newEnumEventCallWorklist(graph)
	}
	entries := make([]*enumEventState, len(graph.blocks))
	entry := int(graph.entry) - 1
	if entry < 0 || entry >= len(entries) {
		return result
	}
	entries[entry] = cloneEnumEventState(initial)
	queue := []enumEventBlockID{graph.entry}
	queued := make([]bool, len(graph.blocks))
	queued[entry] = true
	for len(queue) != 0 {
		blockID := queue[0]
		queue = queue[1:]
		index := int(blockID) - 1
		queued[index] = false
		state := cloneEnumEventState(entries[index])
		block := graph.blocks[index]
		for eventIndex, event := range block.events {
			location := enumEventLocation{block: blockID, index: eventIndex}
			result.before[location] = cloneEnumEventState(state)
			valid := graph.transfer(state, event)
			if event.kind == enumEventPayloadCheck && event.expression != nil {
				current, found := result.access[event.expression]
				result.access[event.expression] = valid && (!found || current)
			}
			if event.kind == enumEventReturn {
				for len(result.values) < len(event.values) {
					result.values = append(result.values, enumAbstractValue{})
				}
				for valueIndex, saved := range event.values {
					result.values[valueIndex], _ = joinEnumAbstractValues(
						result.values[valueIndex], state.saved[saved],
					)
				}
			}
		}
		if len(block.successors) == 0 {
			result.output, _ = joinEnumEventStates(result.output, state)
		}
		for _, edge := range block.successors {
			outgoing := cloneEnumEventState(state)
			for _, test := range edge.tests {
				graph.transfer(outgoing, test)
			}
			target := int(edge.target) - 1
			if target < 0 || target >= len(entries) {
				continue
			}
			joined, changed := joinEnumEventStates(entries[target], outgoing)
			if !changed {
				continue
			}
			entries[target] = joined
			if !queued[target] {
				queue = append(queue, edge.target)
				queued[target] = true
			}
		}
	}
	if graph.calls.analyze() {
		return graph.run(initial)
	}
	return result
}
