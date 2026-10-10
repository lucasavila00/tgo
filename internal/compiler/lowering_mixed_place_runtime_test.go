package compiler

import "testing"

func TestLoweringPreservesMixedGenericPlaceSemantics(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "mixedplaceruntime", sourceName: "mixed_place.tgo",
		source: loweringMixedPlaceRuntimeSource, testName: "mixed_place_test.tgo",
		testSource:  loweringMixedPlaceRuntimeTestSource,
		testPattern: "TestGeneratedMixedPlaces",
	})
}

const loweringMixedPlaceRuntimeSource = `package mixedplaceruntime

import "errors"

var errMixed = errors.New("mixed failure")

type Container interface {
	~[2]int | ~[]int | ~*[2]int
}

func replace[S Container](current *S, next S, value int, fail bool) (int, error) {
	*current = next
	if fail {
		return 0, errMixed
	}
	return value, nil
}

func assign[S Container](current *S, next S, fail bool) error {
	(*current)[0] = replace(current, next, 9, fail)!!
	return nil
}

func add[S Container](current *S, next S) error {
	(*current)[0] += replace(current, next, 4, false)!!
	return nil
}

func increment[S Container](current *S) {
	(*current)[0]++
}

func replaceValue[S Container](current *S, next S, value int) int {
	*current = next
	return value
}

func nativeAdd[S Container](current *S, next S) {
	(*current)[0] += replaceValue(current, next, 4)
}

type Cell struct {
	Value int
	Next *Cell
}

type CellContainer interface {
	~[2]Cell | ~[]Cell | ~*[2]Cell
}

func value(events *[]string, name string, result int, fail bool) (int, error) {
	if events != nil {
		*events = append(*events, name)
	}
	if fail {
		return 0, errMixed
	}
	return result, nil
}

func recursive[S CellContainer](events *[]string, current *S, fail bool) error {
	(*current)[value(events, "index", 0, false)!!].Next.Value =
		value(events, "right", 9, fail)!!
	*events = append(*events, "after")
	return nil
}

func tuple(events *[]string, fail bool) (int, int, error) {
	*events = append(*events, "right")
	if fail {
		return 0, 0, errMixed
	}
	return 7, 9, nil
}

func plainTuple(events *[]string) (int, int) {
	*events = append(*events, "plain-right")
	return 5, 6
}

func twoStores[S Container](
	events *[]string,
	ordinary []int,
	current S,
	fail bool,
) error {
	ordinary[value(events, "ordinary-index", 0, false)!!],
		current[value(events, "mixed-index", 0, false)!!] = tuple(events, fail)!!
	*events = append(*events, "after")
	return nil
}

func reversedStores[S Container](events *[]string, current S, ordinary []int) error {
	current[value(events, "mixed-index", 0, false)!!],
		ordinary[value(events, "ordinary-index", 0, false)!!] = tuple(events, false)!!
	return nil
}

func ordinaryTupleStores[S Container](events *[]string, current S, ordinary []int) error {
	current[value(events, "mixed-index", 0, false)!!],
		ordinary[value(events, "ordinary-index", 0, false)!!] = plainTuple(events)
	return nil
}

type PointerContainer interface {
	~[2]*int | ~[]*int
}

func contextualNil[S PointerContainer](current S) error {
	current[0], _ = nil, value(nil, "unused", 0, false)!!
	return nil
}

type word uint64

type WordContainer interface {
	~[2]word | ~[]word
}

func contextualShift[S WordContainer](current S) error {
	current[0] = 1 << value(nil, "unused", 40, false)!!
	return nil
}

func reflectCollision[S Container](reflect int, current S) error {
	current[0] = value(nil, "unused", reflect, false)!!
	return nil
}

type narrowContainer interface {
	Container
	~[2]int | ~[]int
}

func narrowed[S narrowContainer](current S) error {
	current[0] = value(nil, "unused", 6, false)!!
	return nil
}

type flag bool

type BoolContainer interface {
	~[2]flag | ~[]flag
}

func checkedBool(result bool) (bool, error) {
	return result, nil
}

func contextualBool[S BoolContainer](current S, left int, right int) error {
	current[0] = left < right && checkedBool(true)!!
	return nil
}

func selectedReceive[S Container](events *[]string, current S, input <-chan int) error {
	select {
	case current[value(events, "selected-index", 0, false)!!] = <-input:
		*events = append(*events, "selected-body")
	}
	return nil
}
`

const loweringMixedPlaceRuntimeTestSource = `package mixedplaceruntime

import "testing"

type namedArray [2]int
type namedSlice []int
type namedPointer *[2]int
type namedCellArray [2]Cell
type namedCellPointer *[2]Cell

func TestGeneratedMixedPlaces(t *testing.T) {
	array := namedArray{1, 2}
	if err := assign(&array, namedArray{3, 4}, false); err != nil || array != (namedArray{9, 4}) {
		t.Fatalf("array=%v error=%v", array, err)
	}

	oldSlice := namedSlice{1, 2}
	slice := oldSlice
	newSlice := namedSlice{3, 4}
	if err := assign(&slice, newSlice, false); err != nil || oldSlice[0] != 9 ||
		slice[0] != 3 {
		t.Fatalf("old slice=%v current=%v error=%v", oldSlice, slice, err)
	}

	oldArray := [2]int{1, 2}
	newArray := [2]int{3, 4}
	pointer := namedPointer(&oldArray)
	if err := assign(&pointer, namedPointer(&newArray), false); err != nil ||
		oldArray[0] != 9 || newArray[0] != 3 {
		t.Fatalf("old array=%v new array=%v error=%v", oldArray, newArray, err)
	}

	failing := namedSlice{1, 2}
	replacement := namedSlice{3, 4}
	if err := assign(&failing, replacement, true); err != errMixed ||
		failing[0] != 3 || replacement[0] != 3 {
		t.Fatalf("failed current=%v replacement=%v error=%v", failing, replacement, err)
	}

	array = namedArray{1, 2}
	nativeArray := namedArray{1, 2}
	nativeAdd(&nativeArray, namedArray{3, 4})
	if err := add(&array, namedArray{3, 4}); err != nil || array != nativeArray {
		t.Fatalf("compound array=%v native=%v error=%v", array, nativeArray, err)
	}
	compoundOldSlice := namedSlice{1, 2}
	compoundSlice := compoundOldSlice
	nativeOldSlice := namedSlice{1, 2}
	nativeSlice := nativeOldSlice
	nextSlice := namedSlice{3, 4}
	nativeAdd(&nativeSlice, nextSlice)
	if err := add(&compoundSlice, nextSlice); err != nil ||
		compoundOldSlice[0] != nativeOldSlice[0] || compoundSlice[0] != nativeSlice[0] {
		t.Fatalf("compound slice old=%v current=%v native-old=%v native=%v error=%v",
			compoundOldSlice, compoundSlice, nativeOldSlice, nativeSlice, err)
	}
	compoundOldArray := [2]int{1, 2}
	compoundNewArray := [2]int{3, 4}
	compoundPointer := namedPointer(&compoundOldArray)
	nativeOldArray := [2]int{1, 2}
	nativeNewArray := [2]int{3, 4}
	nativePointer := namedPointer(&nativeOldArray)
	nativeAdd(&nativePointer, namedPointer(&nativeNewArray))
	if err := add(&compoundPointer, namedPointer(&compoundNewArray)); err != nil ||
		compoundOldArray != nativeOldArray || compoundNewArray != nativeNewArray {
		t.Fatalf("compound pointer old=%v next=%v native-old=%v native-next=%v error=%v",
			compoundOldArray, compoundNewArray, nativeOldArray, nativeNewArray, err)
	}
	increment(&array)
	if array[0] != nativeArray[0]+1 {
		t.Fatalf("incremented array=%v", array)
	}

	cell := Cell{Value: 1}
	cells := []Cell{{Next: &cell}}
	events := []string{}
	if err := recursive(&events, &cells, false); err != nil || cell.Value != 9 ||
		len(events) != 3 || events[0] != "index" || events[1] != "right" ||
		events[2] != "after" {
		t.Fatalf("recursive cell=%v events=%v error=%v", cell, events, err)
	}
	cell.Value = 1
	cellArray := namedCellArray{{Next: &cell}}
	events = nil
	if err := recursive(&events, &cellArray, false); err != nil || cell.Value != 9 {
		t.Fatalf("recursive array cell=%v events=%v error=%v", cell, events, err)
	}
	cell.Value = 1
	pointerArray := [2]Cell{{Next: &cell}}
	cellPointer := namedCellPointer(&pointerArray)
	events = nil
	if err := recursive(&events, &cellPointer, false); err != nil || cell.Value != 9 {
		t.Fatalf("recursive pointer cell=%v events=%v error=%v", cell, events, err)
	}

	cell.Value = 1
	events = nil
	if err := recursive(&events, &cells, true); err != errMixed || cell.Value != 1 ||
		len(events) != 2 || events[0] != "index" || events[1] != "right" {
		t.Fatalf("failed recursive cell=%v events=%v error=%v", cell, events, err)
	}

	ordinary := []int{1}
	mixed := namedSlice{2}
	events = nil
	if err := twoStores(&events, ordinary, mixed, false); err != nil ||
		ordinary[0] != 7 || mixed[0] != 9 {
		t.Fatalf("stores ordinary=%v mixed=%v events=%v error=%v",
			ordinary, mixed, events, err)
	}
	ordinary, mixed, events = []int{1}, namedSlice{2}, nil
	if err := reversedStores(&events, mixed, ordinary); err != nil ||
		mixed[0] != 7 || ordinary[0] != 9 {
		t.Fatalf("reversed ordinary=%v mixed=%v events=%v error=%v",
			ordinary, mixed, events, err)
	}
	ordinary, mixed, events = []int{1}, namedSlice{2}, nil
	if err := ordinaryTupleStores(&events, mixed, ordinary); err != nil ||
		mixed[0] != 5 || ordinary[0] != 6 || len(events) != 3 ||
		events[2] != "plain-right" {
		t.Fatalf("plain tuple ordinary=%v mixed=%v events=%v error=%v",
			ordinary, mixed, events, err)
	}

	ordinary, mixed, events = []int{1}, namedSlice{2}, nil
	if err := twoStores(&events, ordinary, mixed, true); err != errMixed ||
		ordinary[0] != 1 || mixed[0] != 2 {
		t.Fatalf("failed stores ordinary=%v mixed=%v events=%v error=%v",
			ordinary, mixed, events, err)
	}

	ordinary, events = []int{1}, nil
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		_ = twoStores(&events, ordinary, namedSlice{}, false)
	}()
	if !panicked || ordinary[0] != 7 {
		t.Fatalf("second store panic=%v ordinary=%v events=%v",
			panicked, ordinary, events)
	}

	pointers := []*int{&ordinary[0]}
	if err := contextualNil(pointers); err != nil || pointers[0] != nil {
		t.Fatalf("contextual nil=%v error=%v", pointers, err)
	}
	words := []word{0}
	if err := contextualShift(words); err != nil || words[0] != 1<<40 {
		t.Fatalf("contextual shift=%v error=%v", words, err)
	}
	if err := reflectCollision(8, mixed); err != nil || mixed[0] != 8 {
		t.Fatalf("reflect collision=%v error=%v", mixed, err)
	}
	if err := narrowed(mixed); err != nil || mixed[0] != 6 {
		t.Fatalf("narrowed=%v error=%v", mixed, err)
	}
	flags := []flag{false}
	if err := contextualBool(flags, 1, 2); err != nil || !flags[0] {
		t.Fatalf("contextual bool=%v error=%v", flags, err)
	}
	input := make(chan int, 1)
	input <- 11
	events = nil
	if err := selectedReceive(&events, mixed, input); err != nil || mixed[0] != 11 ||
		len(events) != 2 || events[0] != "selected-index" ||
		events[1] != "selected-body" {
		t.Fatalf("selected mixed=%v events=%v error=%v", mixed, events, err)
	}
}
`
