package compiler

import "testing"

func TestForInitializerRuntimeOrderAndIdentity(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "loopinit", sourceName: "loop.tgo", source: loopInitializerSource,
		testName: "loop_test.tgo", testSource: loopInitializerTestSource,
		testPattern: "TestGeneratedForInitializer",
	})
}

const loopInitializerSource = `package loopinit

import "errors"

var errInitializer = errors.New("initializer failure")

func record(events *[]string, event string, value int) int {
	*events = append(*events, event)
	return value
}

func load(events *[]string, fail bool) (int, error) {
	record(events, "load", 0)
	if fail {
		return 0, errInitializer
	}
	return 0, nil
}

func combine(events *[]string, left int, value int, right int) int {
	record(events, "combine", 0)
	return left + value + right
}

func condition(events *[]string, allow bool, value int) bool {
	record(events, "condition", 0)
	return allow && value < 3
}

func post(events *[]string, value int) int {
	record(events, "post", 0)
	return value + 1
}

func loop(events *[]string, allow bool, fail bool) ([]func() int, []*int, error) {
	closures := []func() int{}
	addresses := []*int{}
	for value := combine(events,
		record(events, "left", 0),
		load(events, fail)!!,
		record(events, "right", 0));
		condition(events, allow, value);
		value = post(events, value) {
		record(events, "body", 0)
		closures = append(closures, func() int { return value })
		addresses = append(addresses, &value)
	}
	return closures, addresses, nil
}
`

const loopInitializerTestSource = `package loopinit

import (
	"strings"
	"testing"
)

func TestGeneratedForInitializer(t *testing.T) {
	events := []string{}
	closures, addresses, err := loop(&events, false, true)
	if err != errInitializer || closures != nil || addresses != nil ||
		strings.Join(events, ",") != "left,load" {
		t.Fatalf(
			"failure closures=%v addresses=%v error=%v events=%v",
			closures, addresses, err, events,
		)
	}

	events = nil
	closures, addresses, err = loop(&events, true, false)
	wantEvents := "left,load,right,combine,condition,body,post,condition," +
		"body,post,condition,body,post,condition"
	if err != nil || len(closures) != 3 || len(addresses) != 3 ||
		strings.Join(events, ",") != wantEvents {
		t.Fatalf("success closures=%d addresses=%d error=%v events=%v",
			len(closures), len(addresses), err, events)
	}
	for index := range 3 {
		if closures[index]() != index || *addresses[index] != index {
			t.Fatalf(
				"iteration=%d closure=%d address=%d",
				index, closures[index](), *addresses[index],
			)
		}
		for prior := range index {
			if addresses[index] == addresses[prior] {
				t.Fatalf("iterations %d and %d share an address", prior, index)
			}
		}
	}
}
`
