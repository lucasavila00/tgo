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
