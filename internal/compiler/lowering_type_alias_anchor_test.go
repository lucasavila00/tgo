package compiler

import "testing"

func TestLoweringAnchorsShadowedLocalTypeNames(t *testing.T) {
	runLoweringRuntimeTest(t, loweringRuntimeFixture{
		path: "typealiasanchor", sourceName: "type_alias_anchor.tgo",
		source: loweringTypeAliasAnchorSource, testName: "type_alias_anchor_test.tgo",
		testSource:  loweringTypeAliasAnchorTestSource,
		testPattern: "TestGeneratedTypeAliasAnchor",
	})
}

const loweringTypeAliasAnchorSource = `package typealiasanchor

import (
	"errors"
	"reflect"
	. "time"
	"unsafe"
)

var errLoad = errors.New("load failure")

type GenericFlag[T any] bool

func consumeGeneric[V any](value GenericFlag[V], number int) int {
	if value { return number }
	return 0
}

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

func anchored(events *[]string, a int, b int, fail bool) (int, error) {
	type Flag bool
	consume := func(value Flag, number int) int {
		*events = append(*events, reflect.TypeOf(value).Name())
		if value {
			return number
		}
		return 0
	}
	{
		type Flag int
		shadow := Flag(1)
		return consume(
			mark(events, "left", a) < mark(events, "right", b),
			load(events, fail)!!,
		) + int(shadow), nil
	}
}

func shifted(events *[]string) int {
	*events = append(*events, "shift")
	return 1
}

func dotImported(events *[]string, fail bool) (Duration, error) {
	consume := func(value Duration, number int) Duration { return value + Duration(number) }
	Duration := 1
	_ = Duration
	return consume(1 << shifted(events), load(events, fail)!!), nil
}

func loadChannel(events *[]string, channel chan int) (chan int, error) {
	*events = append(*events, "channel")
	return channel, nil
}

func unsafePointer(events *[]string) error {
	channel := make(chan unsafe.Pointer, 1)
	var disabled chan int
	value := 1
	select {
	case channel <- unsafe.Pointer(&value):
	case <-loadChannel(events, disabled)!!:
	}
	return nil
}

func genericAnchor(events *[]string) (int, error) {
	type GenericFlag bool
	type V int
	_ = GenericFlag(false)
	return consumeGeneric[V](
		mark(events, "left", 1) < mark(events, "right", 2),
		load(events, false)!!,
	), nil
}
`

const loweringTypeAliasAnchorTestSource = `package typealiasanchor

import (
	"strings"
	"testing"
)

func TestGeneratedTypeAliasAnchor(t *testing.T) {
	events := []string{}
	value, err := anchored(&events, 1, 2, false)
	if value != 8 || err != nil || strings.Join(events, ",") != "left,right,load,Flag" {
		t.Fatalf("success value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	value, err = anchored(&events, 1, 2, true)
	if value != 0 || err != errLoad || strings.Join(events, ",") != "left,right,load" {
		t.Fatalf("failure value=%d error=%v events=%v", value, err, events)
	}

	events = nil
	duration, err := dotImported(&events, false)
	if duration != 9 || err != nil || strings.Join(events, ",") != "shift,load" {
		t.Fatalf("dot import value=%v error=%v events=%v", duration, err, events)
	}

	events = nil
	if err := unsafePointer(&events); err != nil || strings.Join(events, ",") != "channel" {
		t.Fatalf("unsafe pointer error=%v events=%v", err, events)
	}

	events = nil
	value, err = genericAnchor(&events)
	if value != 7 || err != nil || strings.Join(events, ",") != "left,right,load" {
		t.Fatalf("generic anchor value=%d error=%v events=%v", value, err, events)
	}
}
`
