package tgolint

import "tgo/pkg/syntax"

type enumEventCallID int

type enumEventCallKey struct {
	closure enumEventClosureID
}

type enumEventCallSummary struct {
	inputs         []enumAbstractValue
	captures       map[enumCellID]enumAbstractValue
	captureCells   enumCellSet
	inputState     *enumEventState
	results        []enumAbstractValue
	cellWrites     map[enumCellID]enumAbstractValue
	argumentWrites map[int]bool
	captureWrites  map[enumCellID]bool
	regionWrites   enumRegionSet
	dependents     map[enumEventCallID]bool
	queued         bool
	analyzing      bool
}

type enumEventCallWorklist struct {
	graph     *enumEventGraph
	calls     map[enumEventCallKey]enumEventCallID
	summaries []*enumEventCallSummary
	queue     []enumEventCallID
	changed   bool
}

func newEnumEventCallWorklist(graph *enumEventGraph) *enumEventCallWorklist {
	return &enumEventCallWorklist{
		graph: graph,
		calls: make(map[enumEventCallKey]enumEventCallID),
	}
}

func (worklist *enumEventCallWorklist) call(
	key enumEventCallKey,
) enumEventCallID {
	if id, found := worklist.calls[key]; found {
		return id
	}
	id := enumEventCallID(len(worklist.summaries) + 1)
	worklist.calls[key] = id
	captureCells := enumCellSet(nil)
	index := int(key.closure) - 1
	if index >= 0 && index < len(worklist.graph.identities.closureCaptures) {
		captureCells = cloneEnumCellSet(
			worklist.graph.identities.closureCaptures[index],
		)
	}
	worklist.summaries = append(worklist.summaries, &enumEventCallSummary{
		captures:       make(map[enumCellID]enumAbstractValue),
		captureCells:   captureCells,
		cellWrites:     make(map[enumCellID]enumAbstractValue),
		argumentWrites: make(map[int]bool),
		captureWrites:  make(map[enumCellID]bool),
		regionWrites:   make(enumRegionSet),
		dependents:     make(map[enumEventCallID]bool),
	})
	return id
}

func (worklist *enumEventCallWorklist) summary(
	id enumEventCallID,
) *enumEventCallSummary {
	index := int(id) - 1
	if index < 0 || index >= len(worklist.summaries) {
		return nil
	}
	return worklist.summaries[index]
}

func (worklist *enumEventCallWorklist) addInput(
	id enumEventCallID,
	arguments []enumAbstractValue,
	state *enumEventState,
) bool {
	summary := worklist.summary(id)
	if summary == nil {
		return false
	}
	changed := false
	for len(summary.inputs) < len(arguments) {
		summary.inputs = append(summary.inputs, enumAbstractValue{})
		changed = true
	}
	for index, argument := range arguments {
		joined, added := joinEnumAbstractValues(summary.inputs[index], argument)
		summary.inputs[index] = joined
		changed = changed || added
	}
	for cell := range summary.captureCells {
		capture := state.cells[cell]
		joined, added := joinEnumAbstractValues(summary.captures[cell], capture)
		summary.captures[cell] = joined
		changed = changed || added
	}
	var stateChanged bool
	summary.inputState, stateChanged = joinEnumEventStates(summary.inputState, state)
	changed = changed || stateChanged
	if changed {
		worklist.changed = true
		worklist.enqueue(id)
	}
	return changed
}

func (worklist *enumEventCallWorklist) addOutput(
	id enumEventCallID,
	results []enumAbstractValue,
	cells map[enumCellID]enumAbstractValue,
	arguments map[int]bool,
	captures map[enumCellID]bool,
	regions enumRegionSet,
) bool {
	summary := worklist.summary(id)
	if summary == nil {
		return false
	}
	changed := false
	for len(summary.results) < len(results) {
		summary.results = append(summary.results, enumAbstractValue{})
		changed = true
	}
	for index, result := range results {
		joined, added := joinEnumAbstractValues(summary.results[index], result)
		summary.results[index] = joined
		changed = changed || added
	}
	for cell, value := range cells {
		joined, added := joinEnumAbstractValues(summary.cellWrites[cell], value)
		summary.cellWrites[cell] = joined
		changed = changed || added
	}
	for argument := range arguments {
		if !summary.argumentWrites[argument] {
			summary.argumentWrites[argument] = true
			changed = true
		}
	}
	for capture := range captures {
		if !summary.captureWrites[capture] {
			summary.captureWrites[capture] = true
			changed = true
		}
	}
	for region := range regions {
		if !summary.regionWrites[region] {
			summary.regionWrites[region] = true
			changed = true
		}
	}
	if changed {
		worklist.changed = true
		for dependent := range summary.dependents {
			worklist.enqueue(dependent)
		}
	}
	return changed
}

func (worklist *enumEventCallWorklist) depend(
	callee enumEventCallID,
	caller enumEventCallID,
) {
	summary := worklist.summary(callee)
	if summary != nil && caller != 0 {
		summary.dependents[caller] = true
	}
}

func (worklist *enumEventCallWorklist) enqueue(id enumEventCallID) {
	summary := worklist.summary(id)
	if summary == nil || summary.queued {
		return
	}
	summary.queued = true
	worklist.queue = append(worklist.queue, id)
}

func (worklist *enumEventCallWorklist) next() enumEventCallID {
	if len(worklist.queue) == 0 {
		return 0
	}
	id := worklist.queue[0]
	worklist.queue = worklist.queue[1:]
	if summary := worklist.summary(id); summary != nil {
		summary.queued = false
	}
	return id
}

// applyAlternatives joins closure outputs before it changes the caller state.
func (worklist *enumEventCallWorklist) applyAlternatives(
	state *enumEventState,
	caller enumEventCallID,
	callee enumAbstractValue,
	arguments []enumAbstractValue,
	results []enumSavedValueID,
) {
	joinedResults := make([]enumAbstractValue, len(results))
	joinedCells := make(map[enumCellID]enumAbstractValue)
	joinedRegions := make(enumRegionSet)
	for closure := range callee.closures {
		call := worklist.call(enumEventCallKey{closure: closure})
		inputState := cloneEnumEventState(state)
		for cell, input := range callee.closureInputs[closure] {
			inputState.cells[cell] = cloneEnumAbstractValue(input)
		}
		worklist.addInput(call, arguments, inputState)
		worklist.depend(call, caller)
		summary := worklist.summary(call)
		if summary == nil {
			continue
		}
		for index := range joinedResults {
			if index < len(summary.results) {
				current := enumInstantiateClosureInputs(
					summary.results[index], arguments,
				)
				joinedResults[index], _ = joinEnumAbstractValues(
					joinedResults[index], current,
				)
			}
		}
		for cell, value := range summary.cellWrites {
			joinedCells[cell], _ = joinEnumAbstractValues(joinedCells[cell], value)
		}
		for argument := range summary.argumentWrites {
			if argument < len(arguments) {
				for region := range arguments[argument].regions {
					joinedRegions[region] = true
				}
			}
		}
		for capture := range summary.captureWrites {
			input := state.cells[capture]
			if saved, found := callee.closureInputs[closure][capture]; found {
				input = saved
			}
			for region := range input.regions {
				joinedRegions[region] = true
			}
		}
		for region := range summary.regionWrites {
			joinedRegions[region] = true
		}
	}
	for index, saved := range results {
		state.saved[saved] = joinedResults[index]
	}
	for cell, value := range joinedCells {
		state.recordCellWrite(cell, 0)
		state.cells[cell], _ = joinEnumAbstractValues(state.cells[cell], value)
	}
	for region := range joinedRegions {
		state.killRegionProofs(enumRegionSet{region: true})
		state.recordRegionWrite(region, 0)
	}
	state.invalidateObservations(enumCellsOf(joinedCells), joinedRegions)
}

func (worklist *enumEventCallWorklist) analyze() bool {
	worklist.changed = false
	for call := worklist.next(); call != 0; call = worklist.next() {
		summary := worklist.summary(call)
		if summary == nil {
			continue
		}
		if summary.analyzing {
			worklist.enqueue(call)
			return worklist.changed
		}
		summary.analyzing = true
		worklist.analyzeCall(call, summary)
		summary.analyzing = false
	}
	return worklist.changed
}

func (worklist *enumEventCallWorklist) analyzeCall(
	call enumEventCallID,
	summary *enumEventCallSummary,
) {
	var key enumEventCallKey
	found := false
	for candidate, id := range worklist.calls {
		if id == call {
			key = candidate
			found = true
			break
		}
	}
	closureIndex := int(key.closure) - 1
	if !found || closureIndex < 0 ||
		closureIndex >= len(worklist.graph.identities.closureKeys) {
		return
	}
	literal := worklist.graph.identities.closureKeys[closureIndex].literal
	creation := worklist.graph.identities.closureKeys[closureIndex].activation
	builder := worklist.graph.checker.newEnumClosureEventBuilder(
		worklist.graph, literal, creation,
	)
	initial := enumInvocationState(summary.inputState)
	argument := 0
	for _, field := range literal.Type.Params.List {
		for _, name := range field.Names {
			if argument >= len(summary.inputs) {
				break
			}
			cell := builder.graph.identities.cell(enumCellKey{
				activation: builder.activation,
				object:     worklist.graph.checker.facts.Object(name),
			})
			if syntax.EllipsisExpressionOf(field.Type) != nil {
				value := enumAbstractValue{}
				for ; argument < len(summary.inputs); argument++ {
					value, _ = joinEnumAbstractValues(value, summary.inputs[argument])
				}
				initial.cells[cell] = value
			} else {
				initial.cells[cell] = cloneEnumAbstractValue(summary.inputs[argument])
				argument++
			}
		}
	}
	child := builder.build(literal.Body)
	child.calls = worklist
	child.call = call
	result := child.runPass(initial)
	cells := make(map[enumCellID]enumAbstractValue)
	argumentWrites := make(map[int]bool)
	captureWrites := make(map[enumCellID]bool)
	regions := make(enumRegionSet)
	if result.output != nil {
		for cell := range result.output.cellWrites {
			cells[cell] = cloneEnumAbstractValue(result.output.cells[cell])
		}
		for region := range result.output.regionWrites {
			captured := false
			for cell, capture := range summary.captures {
				if capture.regions[region] {
					captureWrites[cell] = true
					captured = true
				}
			}
			if !captured {
				mapped := false
				for index, input := range summary.inputs {
					if input.regions[region] {
						argumentWrites[index] = true
						mapped = true
					}
				}
				if !mapped {
					regions[region] = true
				}
			}
		}
	}
	formalResults := make([]enumAbstractValue, len(result.values))
	for index, value := range result.values {
		formalResults[index] = enumFormalizeClosureInputs(value, summary.inputs)
	}
	worklist.addOutput(
		call, formalResults, cells, argumentWrites, captureWrites, regions,
	)
}

func enumFormalizeClosureInputs(
	value enumAbstractValue,
	inputs []enumAbstractValue,
) enumAbstractValue {
	value = cloneEnumAbstractValue(value)
	for closure, captures := range value.closureInputs {
		for cell, capture := range captures {
			for index, input := range inputs {
				if enumRegionsOverlap(capture.regions, input.regions) {
					capture.formal = index + 1
					capture.regions = nil
					captures[cell] = capture
					break
				}
			}
		}
		value.closureInputs[closure] = captures
	}
	return value
}

func enumInstantiateClosureInputs(
	value enumAbstractValue,
	arguments []enumAbstractValue,
) enumAbstractValue {
	value = cloneEnumAbstractValue(value)
	for closure, captures := range value.closureInputs {
		for cell, capture := range captures {
			index := capture.formal - 1
			if index >= 0 && index < len(arguments) {
				captures[cell] = cloneEnumAbstractValue(arguments[index])
			}
		}
		value.closureInputs[closure] = captures
	}
	return value
}

func enumRegionsOverlap(left, right enumRegionSet) bool {
	for region := range left {
		if right[region] {
			return true
		}
	}
	return false
}

func enumInvocationState(input *enumEventState) *enumEventState {
	result := cloneEnumEventState(input)
	if result == nil {
		result = newEnumEventState()
	}
	result.cellWrites = make(map[enumCellID]enumWriteSet)
	result.regionWrites = make(map[enumRegionID]enumWriteSet)
	return result
}

func enumCellsOf(values map[enumCellID]enumAbstractValue) enumCellSet {
	result := make(enumCellSet, len(values))
	for cell := range values {
		result[cell] = true
	}
	return result
}
