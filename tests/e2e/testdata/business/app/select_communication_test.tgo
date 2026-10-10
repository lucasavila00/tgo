package app

import (
	"strings"
	"testing"
)

func TestPropagationSelectPreselectionOrder(t *testing.T) {
	events := []string{}
	value, err := PropagationSelectReceiveOrder(&events)
	if value != 7 || err != nil ||
		strings.Join(events, ",") != "send-channel,send-value,receive-channel,receive-body" {
		t.Fatalf("receive value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	value, err = PropagationSelectSendOrder(&events)
	if value != 5 || err != nil ||
		strings.Join(events, ",") != "send-channel,send-value,receive-channel,send-body" {
		t.Fatalf("send value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	err = PropagationSelectDefault(&events)
	if err != nil || strings.Join(events, ",") != "send-channel,send-value,receive-channel,default" {
		t.Fatalf("default error=%v events=%v", err, events)
	}
	events = nil
	value, err = PropagationSelectOperandSnapshot(&events)
	if value != 6 || err != nil || strings.Join(events, ",") != "send-value,replace,send-body" {
		t.Fatalf("snapshot value=%d error=%v events=%v", value, err, events)
	}
	events = nil
	narrow, err := PropagationSelectUntypedSend(&events)
	if narrow != 1 || err != nil || strings.Join(events, ",") != "receive-channel" {
		t.Fatalf("untyped value=%d error=%v events=%v", narrow, err, events)
	}
}

func TestPropagationSelectPreselectionErrors(t *testing.T) {
	tests := []struct {
		failure string
		want    string
	}{
		{failure: "channel", want: "send-channel"},
		{failure: "value", want: "send-channel,send-value"},
		{failure: "receive", want: "send-channel,send-value,receive-channel"},
	}
	for _, test := range tests {
		t.Run(test.failure, func(t *testing.T) {
			events := []string{}
			err := PropagationSelectPreFailure(&events, test.failure)
			if err != errSelectCommunication || strings.Join(events, ",") != test.want {
				t.Fatalf("error=%v events=%v", err, events)
			}
		})
	}
	events := []string{}
	err := PropagationSelectWrapped(&events)
	if err == nil || err == errSelectCommunication ||
		err.Error() != "selectValue: select communication failure" {
		t.Fatalf("wrapped error=%v", err)
	}
}

func TestPropagationSelectReceiveTargets(t *testing.T) {
	values := [1]int{}
	statuses := [1]bool{}
	events := []string{}
	err := PropagationSelectTargets(&events, &values, &statuses, false, "")
	if err != nil || values[0] != 0 || statuses[0] ||
		strings.Join(events, ",") != "receive-channel,default" {
		t.Fatalf("default values=%v statuses=%v error=%v events=%v", values, statuses, err, events)
	}
	events = nil
	err = PropagationSelectTargets(&events, &values, &statuses, true, "")
	if err != nil || values[0] != 9 || !statuses[0] ||
		strings.Join(events, ",") != "receive-channel,left,right,body" {
		t.Fatalf("receive values=%v statuses=%v error=%v events=%v", values, statuses, err, events)
	}
	events = nil
	err = PropagationSelectTargets(&events, &values, &statuses, true, "left")
	if err != errSelectCommunication || strings.Join(events, ",") != "receive-channel,left" {
		t.Fatalf("target error=%v events=%v", err, events)
	}
}

func TestPropagationSelectControlAndScope(t *testing.T) {
	events := []string{}
	err := PropagationSelectGotoBreak(&events)
	if err != nil || strings.Join(events, ",") != "receive-channel,body,done" {
		t.Fatalf("goto break error=%v events=%v", err, events)
	}
	events = nil
	err = PropagationSelectContinue(&events)
	if err != nil || strings.Join(events, ",") != "receive-channel,body,receive-channel,body" {
		t.Fatalf("continue error=%v events=%v", err, events)
	}
	events = nil
	value, ok, err := PropagationSelectDeclaration(&events)
	if value != 4 || !ok || err != nil || strings.Join(events, ",") != "receive-channel" {
		t.Fatalf("declaration value=%d ok=%v error=%v events=%v", value, ok, err, events)
	}
}
