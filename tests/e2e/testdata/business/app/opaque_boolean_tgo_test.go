package app

import (
	"strings"
	"testing"

	"example.com/business/opaquebool"
)

func TestOpaqueBooleanKeepsForeignDynamicType(t *testing.T) {
	events := []string{}
	ok, err := OpaqueBooleanFactory(&events, false)
	if !ok || err != nil || strings.Join(events, ",") != "later,consume" {
		t.Fatalf("factory ok=%v error=%v events=%v", ok, err, events)
	}

	events = nil
	ok, err = OpaqueBooleanLogical(&events, false)
	if !ok || err != nil || strings.Join(events, ",") != "later,consume" {
		t.Fatalf("logical ok=%v error=%v events=%v", ok, err, events)
	}

	events = nil
	ok, err = OpaqueBooleanChecked(&events, false, false)
	if !ok || err != nil || strings.Join(events, ",") != "later,consume" {
		t.Fatalf("checked ok=%v error=%v events=%v", ok, err, events)
	}
}

func TestOpaqueBooleanPropagationSkipsLaterWork(t *testing.T) {
	events := []string{}
	ok, err := OpaqueBooleanChecked(&events, true, false)
	if ok || err != opaquebool.ErrCheck || len(events) != 0 {
		t.Fatalf("check failure ok=%v error=%v events=%v", ok, err, events)
	}

	events = nil
	ok, err = OpaqueBooleanChecked(&events, false, true)
	if ok || err != errOpaqueBooleanLater || strings.Join(events, ",") != "later" {
		t.Fatalf("later failure ok=%v error=%v events=%v", ok, err, events)
	}
}
