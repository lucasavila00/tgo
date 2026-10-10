package compiler

import "testing"

func TestLoweringPreservesLexicalTypeReferences(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "lexicalreference", sourceName: "lexical_reference.tgo",
		source: loweringLexicalReferenceSource, testName: "lexical_reference_test.tgo",
		testSource:  loweringLexicalReferenceTestSource,
		testPattern: "TestGeneratedLexicalTypeReferences",
	})
}

const loweringLexicalReferenceSource = `package lexicalreference

import "errors"

var errLoad = errors.New("load failure")

func mark(events *[]string, event string, value int) int {
	*events = append(*events, event)
	return value
}

func load(events *[]string, fail bool) (int, error) {
	*events = append(*events, "load")
	if fail {
		return 0, errLoad
	}
	return 7, nil
}

func localFlag(events *[]string, a int, b int, fail bool) (int, error) {
	type Flag bool
	consume := func(value Flag, number int) int {
		*events = append(*events, "consume")
		if value {
			return number
		}
		return 0
	}
	{
		Flag := 1
		return consume(
			mark(events, "left", a) < mark(events, "right", b),
			load(events, fail)!!,
		) + Flag, nil
	}
}

func localAnonymousStruct(events *[]string, fail bool) (int, error) {
	type T int
	makeValue := func() struct{ Value T } {
		*events = append(*events, "make")
		return struct{ Value T }{Value: 3}
	}
	consume := func(value struct{ Value T }, number int) int {
		*events = append(*events, "consume")
		return int(value.Value) + number
	}
	{
		T := 1
		return consume(makeValue(), load(events, fail)!!) + T, nil
	}
}

`

const loweringLexicalReferenceTestSource = `package lexicalreference

import (
	"strings"
	"testing"
)

func TestGeneratedLexicalTypeReferences(t *testing.T) {
	events := []string{}
	value, err := localFlag(&events, 1, 2, false)
	if value != 8 || err != nil || strings.Join(events, ",") != "left,right,load,consume" {
		t.Fatalf("flag success value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	value, err = localFlag(&events, 1, 2, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "left,right,load" {
		t.Fatalf("flag failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = localAnonymousStruct(&events, false)
	if value != 11 || err != nil || strings.Join(events, ",") != "make,load,consume" {
		t.Fatalf("struct success value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	value, err = localAnonymousStruct(&events, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "make,load" {
		t.Fatalf("struct failure value=%d error=%v events=%v", value, err, events)
	}
}
`
