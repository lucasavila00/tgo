package compiler

import "testing"

func TestLoweringReturnsGenericZeroResults(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "zeroresult", sourceName: "zero_result.tgo",
		source: loweringZeroResultSource, testName: "zero_result_test.tgo",
		testSource:  loweringZeroResultTestSource,
		testPattern: "TestGeneratedGenericZeroResults",
	})
}

const loweringZeroResultSource = `package zeroresult

import "errors"

type record struct {
	Number int
}

var errLoad = errors.New("load failure")

func addEvent(events *[]string, event string) {
	*events = append(*events, event)
}

func load[T any](events *[]string, value T, fail bool) (T, error) {
	addEvent(events, "load")
	if fail {
		return value, errLoad
	}
	return value, nil
}

func propagated[T any](events *[]string, value T, fail bool) (T, error) {
	defer addEvent(events, "defer")
	return load(events, value, fail)!!, nil
}
`

const loweringZeroResultTestSource = `package zeroresult

import (
	"strings"
	"testing"
)

func checkEvents(t *testing.T, events []string) {
	t.Helper()
	if strings.Join(events, ",") != "load,defer" {
		t.Fatalf("events=%v", events)
	}
}

func TestGeneratedGenericZeroResults(t *testing.T) {
	events := []string{}
	integer, err := propagated(&events, 7, false)
	if integer != 7 || err != nil {
		t.Fatalf("integer success value=%d error=%v", integer, err)
	}
	checkEvents(t, events)
	events = nil
	integer, err = propagated(&events, 7, true)
	if integer != 0 || err != errLoad {
		t.Fatalf("integer failure value=%d error=%v", integer, err)
	}
	checkEvents(t, events)

	events = nil
	structure, err := propagated(&events, record{Number: 8}, false)
	if structure != (record{Number: 8}) || err != nil {
		t.Fatalf("struct success value=%v error=%v", structure, err)
	}
	checkEvents(t, events)
	events = nil
	structure, err = propagated(&events, record{Number: 8}, true)
	if structure != (record{}) || err != errLoad {
		t.Fatalf("struct failure value=%v error=%v", structure, err)
	}
	checkEvents(t, events)

	events = nil
	original := &record{Number: 9}
	pointer, err := propagated(&events, original, false)
	if pointer != original || err != nil {
		t.Fatalf("pointer success value=%p original=%p error=%v", pointer, original, err)
	}
	checkEvents(t, events)
	events = nil
	pointer, err = propagated(&events, original, true)
	if pointer != nil || err != errLoad {
		t.Fatalf("pointer failure value=%p error=%v", pointer, err)
	}
	checkEvents(t, events)

	events = nil
	originalSlice := []int{10}
	slice, err := propagated(&events, originalSlice, false)
	if len(slice) != 1 || &slice[0] != &originalSlice[0] || err != nil {
		t.Fatalf("slice success value=%v error=%v", slice, err)
	}
	checkEvents(t, events)
	events = nil
	slice, err = propagated(&events, originalSlice, true)
	if slice != nil || err != errLoad {
		t.Fatalf("slice failure value=%v error=%v", slice, err)
	}
	checkEvents(t, events)
}
`
