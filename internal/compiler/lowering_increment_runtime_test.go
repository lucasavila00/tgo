package compiler

import "testing"

func TestLoweringIncrementRuntimeOrder(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "incrementruntime", sourceName: "increment.tgo",
		source: loweringIncrementRuntimeSource, testName: "increment_test.tgo",
		testSource:  loweringIncrementRuntimeTestSource,
		testPattern: "TestGeneratedIncrementRuntime",
	})
}

const loweringIncrementRuntimeSource = `package incrementruntime

import "errors"

var errLoad = errors.New("load failure")

func load(events *[]string, values []int) ([]int, error) {
	*events = append(*events, "load")
	if values[0] < 0 {
		return nil, errLoad
	}
	return values, nil
}

func index(events *[]string) int {
	*events = append(*events, "index")
	return 0
}

func update(events *[]string, values []int) error {
	load(events, values)!![index(events)]++
	return nil
}
`

const loweringIncrementRuntimeTestSource = `package incrementruntime

import (
	"strings"
	"testing"
)

func TestGeneratedIncrementRuntime(t *testing.T) {
	events := []string{}
	values := []int{4}
	err := update(&events, values)
	if err != nil || values[0] != 5 || strings.Join(events, ",") != "load,index" {
		t.Fatalf("success values=%v error=%v events=%v", values, err, events)
	}

	events = nil
	values = []int{-1}
	err = update(&events, values)
	if err != errLoad || values[0] != -1 || strings.Join(events, ",") != "load" {
		t.Fatalf("failure values=%v error=%v events=%v", values, err, events)
	}
}
`
