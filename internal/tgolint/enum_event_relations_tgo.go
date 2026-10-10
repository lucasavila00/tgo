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

func equalEnumAbstractValues(left, right enumAbstractValue) bool {
	return enumRegionSetEqual(left.regions, right.regions) &&
		enumCellSetEqual(left.places, right.places) &&
		enumCellSetEqual(left.relations, right.relations) &&
		left.relationKnown == right.relationKnown &&
		enumClosureSetEqual(left.closures, right.closures) &&
		enumCellSetEqual(left.dependencies, right.dependencies) &&
		enumCellSetEqual(left.readCells, right.readCells) &&
		equalEnumTagObservation(left.observation, right.observation) &&
		equalEnumClosureInputs(left.closureInputs, right.closureInputs) &&
		left.formal == right.formal &&
		left.unknown == right.unknown
}

func equalEnumClosureInputs(
	left map[enumEventClosureID]map[enumCellID]enumAbstractValue,
	right map[enumEventClosureID]map[enumCellID]enumAbstractValue,
) bool {
	if len(left) != len(right) {
		return false
	}
	for closure, leftInputs := range left {
		rightInputs, found := right[closure]
		if !found || len(leftInputs) != len(rightInputs) {
			return false
		}
		for cell, leftInput := range leftInputs {
			rightInput, found := rightInputs[cell]
			if !found || !equalEnumAbstractValues(leftInput, rightInput) {
				return false
			}
		}
	}
	return true
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
