package compiler

import "testing"

func TestLoweringPreservesNamedPlaceRuntimeIdentity(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "namedplaceruntime", sourceName: "named_place.tgo",
		source: loweringNamedPlaceRuntimeSource, testName: "named_place_test.tgo",
		testSource:  loweringNamedPlaceRuntimeTestSource,
		testPattern: "TestGeneratedNamedPlaceRuntime",
	})
}

const loweringNamedPlaceRuntimeSource = `package namedplaceruntime

import "errors"

var errPlace = errors.New("place failure")

type Cell struct {
	Value int
}

type P *Cell
type A *[1]int

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func mutation[T any](
	events *[]string,
	target *T,
	next T,
	value int,
	fail bool,
) (int, error) {
	record(events, "mutation")
	*target = next
	if fail {
		return 0, errPlace
	}
	return value, nil
}

func pointerField(events *[]string, value P, next P, fail bool) error {
	value.Value = mutation(events, &value, next, 9, fail)!!
	record(events, "after")
	return nil
}

func arrayBound(events *[]string, value *A, next A, fail bool) (int, error) {
	return mutation(events, value, next, 0, fail)
}

func pointerArray(events *[]string, value A, next A, fail bool) error {
	value[arrayBound(events, &value, next, fail)!!] = 9
	record(events, "after")
	return nil
}

func sliceValue[S ~[]int](
	events *[]string,
	value *S,
	next S,
	fail bool,
) (int, error) {
	return mutation(events, value, next, 9, fail)
}

func genericSlice[S ~[]int](events *[]string, value *S, next S, fail bool) error {
	(*value)[0] = sliceValue(events, value, next, fail)!!
	record(events, "after")
	return nil
}

func mapValue[M ~map[int]int](
	events *[]string,
	value *M,
	next M,
	fail bool,
) (int, error) {
	return mutation(events, value, next, 9, fail)
}

func genericMap[M ~map[int]int](events *[]string, value *M, next M, fail bool) error {
	(*value)[0] = mapValue(events, value, next, fail)!!
	record(events, "after")
	return nil
}
`

const loweringNamedPlaceRuntimeTestSource = `package namedplaceruntime

import (
	"strings"
	"testing"
)

type namedSlice []int
type namedMap map[int]int

func checkEvents(t *testing.T, events []string, want string) {
	t.Helper()
	if strings.Join(events, ",") != want {
		t.Fatalf("events=%v want=%q", events, want)
	}
}

func TestGeneratedNamedPlaceRuntime(t *testing.T) {
	firstCell := Cell{Value: 1}
	secondCell := Cell{Value: 2}
	events := []string{}
	err := pointerField(&events, P(&firstCell), P(&secondCell), false)
	if err != nil || firstCell.Value != 9 || secondCell.Value != 2 {
		t.Fatalf("pointer field first=%v second=%v error=%v", firstCell, secondCell, err)
	}
	checkEvents(t, events, "mutation,after")

	firstCell.Value = 1
	events = nil
	err = pointerField(&events, P(&firstCell), P(&secondCell), true)
	if err != errPlace || firstCell.Value != 1 || secondCell.Value != 2 {
		t.Fatalf("failed pointer field first=%v second=%v error=%v", firstCell, secondCell, err)
	}
	checkEvents(t, events, "mutation")

	firstArray := [1]int{1}
	secondArray := [1]int{2}
	events = nil
	err = pointerArray(&events, A(&firstArray), A(&secondArray), false)
	if err != nil || firstArray[0] != 9 || secondArray[0] != 2 {
		t.Fatalf("pointer array first=%v second=%v error=%v", firstArray, secondArray, err)
	}
	checkEvents(t, events, "mutation,after")

	firstArray[0] = 1
	events = nil
	err = pointerArray(&events, A(&firstArray), A(&secondArray), true)
	if err != errPlace || firstArray[0] != 1 || secondArray[0] != 2 {
		t.Fatalf("failed pointer array first=%v second=%v error=%v", firstArray, secondArray, err)
	}
	checkEvents(t, events, "mutation")

	firstSlice := namedSlice{1}
	secondSlice := namedSlice{2}
	currentSlice := firstSlice
	events = nil
	err = genericSlice(&events, &currentSlice, secondSlice, false)
	if err != nil || firstSlice[0] != 9 || secondSlice[0] != 2 ||
		&currentSlice[0] != &secondSlice[0] {
		t.Fatalf("slice first=%v second=%v current=%v error=%v",
			firstSlice, secondSlice, currentSlice, err)
	}
	checkEvents(t, events, "mutation,after")

	firstSlice = namedSlice{1}
	secondSlice = namedSlice{2}
	currentSlice = firstSlice
	events = nil
	err = genericSlice(&events, &currentSlice, secondSlice, true)
	if err != errPlace || firstSlice[0] != 1 || secondSlice[0] != 2 ||
		&currentSlice[0] != &secondSlice[0] {
		t.Fatalf("failed slice first=%v second=%v current=%v error=%v",
			firstSlice, secondSlice, currentSlice, err)
	}
	checkEvents(t, events, "mutation")

	firstMap := namedMap{0: 1}
	secondMap := namedMap{0: 2}
	currentMap := firstMap
	events = nil
	err = genericMap(&events, &currentMap, secondMap, false)
	if err != nil || firstMap[0] != 9 || secondMap[0] != 2 || currentMap[0] != 2 {
		t.Fatalf("map first=%v second=%v current=%v error=%v",
			firstMap, secondMap, currentMap, err)
	}
	checkEvents(t, events, "mutation,after")

	firstMap = namedMap{0: 1}
	secondMap = namedMap{0: 2}
	currentMap = firstMap
	events = nil
	err = genericMap(&events, &currentMap, secondMap, true)
	if err != errPlace || firstMap[0] != 1 || secondMap[0] != 2 || currentMap[0] != 2 {
		t.Fatalf("failed map first=%v second=%v current=%v error=%v",
			firstMap, secondMap, currentMap, err)
	}
	checkEvents(t, events, "mutation")
}
`
