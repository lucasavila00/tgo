package compiler

import "testing"

func TestLoweringPreservesContextualTypesAndPlaces(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "typesplaces", sourceName: "types_places.tgo",
		source: loweringTypesPlacesSource, testName: "types_places_test.tgo",
		testSource:  loweringTypesPlacesTestSource,
		testPattern: "TestGeneratedTypesAndPlaces",
	})
}

const loweringTypesPlacesSource = `package typesplaces

func record(events *[]string, event string) {
	*events = append(*events, event)
}

func shift(events *[]string) (uint, error) {
	record(events, "shift")
	return 63, nil
}

func contextualSend(events *[]string) (uint64, error) {
	values := make(chan uint64, 1)
	select {
	case values <- 1 << shift(events)!!:
	}
	return <-values, nil
}

func offset(events *[]string) (int, error) {
	record(events, "offset")
	return 2, nil
}

func contextualShiftArgument(events *[]string, n int) (uint64, error) {
	consume := func(value uint64, number int) uint64 { return value + uint64(number) }
	return consume(1 << (n + offset(events)!!), loadValue(events)!!), nil
}

func mutate(events *[]string, value *uint) (chan int, error) {
	record(events, "mutate")
	*value = 63
	return nil, nil
}

func snapshottedSend(events *[]string) (int, uint, error) {
	values := make(chan int, 1)
	var n uint = 1
	select {
	case values <- 1 << n:
	case <-mutate(events, &n)!!:
	}
	return <-values, n, nil
}

type cell struct {
	Value int
}

type holder struct {
	Cells [1]cell
}

type promotedHolder struct {
	*cell
}

type promotedMiddle struct {
	*cell
}

type promotedOuter struct {
	promotedMiddle
	cell int
}

func selectHolder(events *[]string, value *holder) *holder {
	record(events, "target")
	return value
}

func selectIndex(events *[]string) int {
	record(events, "index")
	return 0
}

func loadValue(events *[]string) (int, error) {
	record(events, "rhs")
	return 9, nil
}

func storeDirectPlace(events *[]string, value *cell) error {
	value.Value = loadValue(events)!!
	return nil
}

func storeNestedPlace(events *[]string, value *holder) error {
	selectHolder(events, value).Cells[selectIndex(events)].Value = loadValue(events)!!
	return nil
}

func replacePromoted(events *[]string, value *promotedHolder, next *cell) (int, error) {
	record(events, "rhs")
	value.cell = next
	return 11, nil
}

func storePromotedPlace(events *[]string, value *promotedHolder, next *cell) error {
	value.Value = replacePromoted(events, value, next)!!
	return nil
}

func storeDeepPromotedPlace(events *[]string, value *promotedOuter) error {
	value.Value = loadValue(events)!!
	return nil
}

func promotedFactory(events *[]string, value *promotedOuter) *promotedOuter {
	record(events, "factory")
	return value
}

func storeFactoryPromotedPlace(events *[]string, value *promotedOuter) error {
	promotedFactory(events, value).Value = loadValue(events)!!
	return nil
}
`

const loweringTypesPlacesTestSource = `package typesplaces

import (
	"strings"
	"testing"
)

func TestGeneratedTypesAndPlaces(t *testing.T) {
	events := []string{}
	value, err := contextualSend(&events)
	if value != uint64(1)<<63 || err != nil || strings.Join(events, ",") != "shift" {
		t.Fatalf("contextual value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	shifted, err := contextualShiftArgument(&events, 1)
	if shifted != 17 || err != nil || strings.Join(events, ",") != "offset,rhs" {
		t.Fatalf("shift argument=%d error=%v events=%v", shifted, err, events)
	}

	events = nil
	snapshot, n, err := snapshottedSend(&events)
	if snapshot != 2 || n != 63 || err != nil || strings.Join(events, ",") != "mutate" {
		t.Fatalf("snapshot value=%d n=%d error=%v events=%v", snapshot, n, err, events)
	}

	events = nil
	direct := cell{}
	err = storeDirectPlace(&events, &direct)
	if direct.Value != 9 || err != nil || strings.Join(events, ",") != "rhs" {
		t.Fatalf("direct place=%v error=%v events=%v", direct, err, events)
	}

	events = nil
	target := holder{}
	err = storeNestedPlace(&events, &target)
	if target.Cells[0].Value != 9 || err != nil || strings.Join(events, ",") != "target,index,rhs" {
		t.Fatalf("place target=%v error=%v events=%v", target, err, events)
	}

	events = nil
	oldCell := &cell{}
	newCell := &cell{}
	promoted := promotedHolder{cell: oldCell}
	err = storePromotedPlace(&events, &promoted, newCell)
	if err != nil || oldCell.Value != 11 || newCell.Value != 0 ||
		strings.Join(events, ",") != "rhs" {
		t.Fatalf("promoted old=%v new=%v error=%v events=%v", oldCell, newCell, err, events)
	}

	events = nil
	deepCell := &cell{}
	deep := promotedOuter{promotedMiddle: promotedMiddle{cell: deepCell}}
	err = storeDeepPromotedPlace(&events, &deep)
	if err != nil || deepCell.Value != 9 || deep.cell != 0 ||
		strings.Join(events, ",") != "rhs" {
		t.Fatalf("deep promoted=%v direct=%d error=%v events=%v", deepCell, deep.cell, err, events)
	}

	events = nil
	deepCell = &cell{}
	deep = promotedOuter{promotedMiddle: promotedMiddle{cell: deepCell}}
	err = storeFactoryPromotedPlace(&events, &deep)
	if err != nil || deepCell.Value != 9 || strings.Join(events, ",") != "factory,rhs" {
		t.Fatalf("factory promoted=%v error=%v events=%v", deepCell, err, events)
	}
}
`
