package tgolint

import (
	"sort"
	"strconv"
	"strings"
)

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

func enumClosureSignature(closures enumEventClosureSet) string {
	values := make([]int, 0, len(closures))
	for closure := range closures {
		values = append(values, int(closure))
	}
	sort.Ints(values)
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func enumAbstractValueSignature(value enumAbstractValue) string {
	return strings.Join([]string{
		enumRegionSignature(value.regions),
		enumCellSignature(value.places),
		enumCellSignature(value.relations),
		strconv.FormatBool(value.relationKnown),
		enumCellSignature(value.readCells),
		enumClosureSignature(value.closures),
		strconv.FormatBool(value.unknown),
	}, "/")
}

func enumCallContext(
	arguments []enumAbstractValue,
	state *enumEventState,
) string {
	parts := make([]string, 0, len(arguments)+len(state.cells))
	for _, argument := range arguments {
		parts = append(parts, "a:"+enumAbstractValueSignature(argument))
	}
	cells := make([]int, 0, len(state.cells))
	for cell := range state.cells {
		cells = append(cells, int(cell))
	}
	sort.Ints(cells)
	for _, raw := range cells {
		cell := enumCellID(raw)
		parts = append(parts, "c:"+strconv.Itoa(raw)+":"+
			enumAbstractValueSignature(state.cells[cell]))
	}
	return strings.Join(parts, "|")
}

func equalEnumAbstractValues(left, right enumAbstractValue) bool {
	return enumRegionSetEqual(left.regions, right.regions) &&
		enumCellSetEqual(left.places, right.places) &&
		enumCellSetEqual(left.relations, right.relations) &&
		left.relationKnown == right.relationKnown &&
		enumClosureSetEqual(left.closures, right.closures) &&
		enumCellSetEqual(left.dependencies, right.dependencies) &&
		enumCellSetEqual(left.readCells, right.readCells) &&
		equalEnumTagObservation(left.observation, right.observation) &&
		left.unknown == right.unknown
}

func enumRegionSetEqual(left, right enumRegionSet) bool {
	if len(left) != len(right) {
		return false
	}
	for region := range left {
		if !right[region] {
			return false
		}
	}
	return true
}

func enumClosureSetEqual(left, right enumEventClosureSet) bool {
	if len(left) != len(right) {
		return false
	}
	for closure := range left {
		if !right[closure] {
			return false
		}
	}
	return true
}
