package compiler

import "testing"

func TestLoweringPreservesNamedLogicalType(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "logicaltype", sourceName: "logical_type.tgo",
		source: loweringLogicalTypeSource, testName: "logical_type_test.tgo",
		testSource:  loweringLogicalTypeTestSource,
		testPattern: "TestGeneratedNamedLogicalType",
	})
}

const loweringLogicalTypeSource = `package logicaltype

import "errors"

type Flag bool

var errCheck = errors.New("check failure")
var errLoad = errors.New("load failure")

func check(events *[]string, value Flag, fail bool) (Flag, error) {
	*events = append(*events, "check")
	if fail {
		return false, errCheck
	}
	return value, nil
}

func load(events *[]string, fail bool) (int, error) {
	*events = append(*events, "load")
	if fail {
		return 0, errLoad
	}
	return 7, nil
}

func consume(events *[]string, value Flag, number int) int {
	*events = append(*events, "consume")
	if value {
		return number
	}
	return 0
}

func use(events *[]string, a int, b int, checkFail bool, loadFail bool) (int, error) {
	return consume(
		events,
		a < b && check(events, true, checkFail)!!,
		load(events, loadFail)!!,
	), nil
}

func preserveDynamicType(events *[]string, loadFail bool) (bool, error) {
	return consumeAny(
		check(events, true, false)!!,
		load(events, loadFail)!!,
	), nil
}

func consumeAny(value any, number int) bool {
	_, ok := value.(Flag)
	return ok && number == 7
}
`

const loweringLogicalTypeTestSource = `package logicaltype

import (
	"strings"
	"testing"
)

func TestGeneratedNamedLogicalType(t *testing.T) {
	events := []string{}
	value, err := use(&events, 2, 1, false, false)
	if value != 0 || err != nil || strings.Join(events, ",") != "load,consume" {
		t.Fatalf("short circuit value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, false, false)
	if value != 7 || err != nil || strings.Join(events, ",") != "check,load,consume" {
		t.Fatalf("success value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, true, false)
	if value != 0 || err != errCheck || strings.Join(events, ",") != "check" {
		t.Fatalf("check failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = use(&events, 1, 2, false, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "check,load" {
		t.Fatalf("load failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	preserved, err := preserveDynamicType(&events, false)
	if !preserved || err != nil || strings.Join(events, ",") != "check,load" {
		t.Fatalf("dynamic type preserved=%t error=%v events=%v", preserved, err, events)
	}

	events = nil
	preserved, err = preserveDynamicType(&events, true)
	if preserved || err != errLoad || strings.Join(events, ",") != "check,load" {
		t.Fatalf("dynamic type failure preserved=%t error=%v events=%v", preserved, err, events)
	}
}
`
