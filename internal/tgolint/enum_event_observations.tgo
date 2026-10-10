package tgolint

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
