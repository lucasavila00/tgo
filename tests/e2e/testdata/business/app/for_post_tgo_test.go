package app

import (
	"strings"
	"testing"
)

func TestPropagationForPostOrderAndErrors(t *testing.T) {
	events := []string{}
	err := PropagationForPostOrder(&events, -1, -1)
	if err != nil || strings.Join(events, ",") != "left,right,left,right" {
		t.Fatalf("success events=%v error=%v", events, err)
	}
	events = nil
	err = PropagationForPostOrder(&events, 0, -1)
	if err != errForPost || strings.Join(events, ",") != "left" {
		t.Fatalf("left failure events=%v error=%v", events, err)
	}
	events = nil
	err = PropagationForPostOrder(&events, -1, 0)
	if err != errForPost || strings.Join(events, ",") != "left,right" {
		t.Fatalf("right failure events=%v error=%v", events, err)
	}
	err = PropagationForPostWrapped()
	if err == nil || err == errForPost || err.Error() != "forPostStep: for post failure" {
		t.Fatalf("wrapped error=%v", err)
	}
	events = nil
	err = PropagationForPostFailureSkipsCondition(&events)
	if err != errForPost || strings.Join(events, ",") != "condition,post" {
		t.Fatalf("condition failure events=%v error=%v", events, err)
	}
}

func TestPropagationForPostControlFlow(t *testing.T) {
	events, err := PropagationForPostContinues()
	if err != nil || strings.Join(events, ",") != "inner,outer,inner,outer" {
		t.Fatalf("continue events=%v error=%v", events, err)
	}
	for _, mode := range []string{"break", "return", "goto"} {
		events, err = PropagationForPostSkipsAbrupt(mode)
		if err != nil || len(events) != 0 {
			t.Fatalf("%s events=%v error=%v", mode, events, err)
		}
	}
	events, err = PropagationForPostGotoContinue()
	if err != nil || strings.Join(events, ",") != "outer,outer" {
		t.Fatalf("goto continue events=%v error=%v", events, err)
	}
}

func TestPropagationForPostIterationIdentity(t *testing.T) {
	valid, err := PropagationForPostIterationIdentity()
	if err != nil || !valid {
		t.Fatalf("valid=%v error=%v", valid, err)
	}
}

func TestPropagationForPostRunsNamedDefer(t *testing.T) {
	events := []string{}
	value, err := PropagationForPostNamedDefer(&events)
	if value != 1 || err != errForPost || strings.Join(events, ",") != "post,defer" {
		t.Fatalf("value=%d error=%v events=%v", value, err, events)
	}
}
