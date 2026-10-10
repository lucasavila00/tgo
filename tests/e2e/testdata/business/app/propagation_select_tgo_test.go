package app

import (
	"errors"
	"testing"
)

func TestPropagationSelectCommunication(t *testing.T) {
	value, err := PropagationSelect(7, true, false)
	if value != 7 || err != nil {
		t.Fatalf("selected value=%d error=%v", value, err)
	}

	value, err = PropagationSelect(7, false, false)
	if value != 0 || err != nil {
		t.Fatalf("default value=%d error=%v", value, err)
	}

	value, err = PropagationSelect(7, true, true)
	if value != 0 || !errors.Is(err, errPropagationSelect) || err == errPropagationSelect {
		t.Fatalf("failed value=%d error=%v", value, err)
	}
	if err.Error() != "loadSelectChannel: select channel failure" {
		t.Fatalf("failed error=%q", err)
	}
}
