package failurereturn

import (
	"errors"
	"testing"
)

func TestFailureReturns(t *testing.T) {
	want := errors.New("failure")
	flag, integer, decimal, complexValue, text, err := Scalars(want)
	if flag || integer != 0 || decimal != 0 || complexValue != 0 || text != "" || err != want {
		t.Fatalf("Scalars = %v, %d, %v, %v, %q, %v", flag, integer, decimal, complexValue, text, err)
	}
	pointer, slice, mapping, channel, function, dynamic, err := References(want)
	if pointer != nil || slice != nil || mapping != nil || channel != nil || function != nil || dynamic != nil || err != want {
		t.Fatalf("References returned a nonzero value")
	}
	item, array, named, err := Values(want)
	if item != (record{}) || array != ([2]int{}) || named != 0 || err != want {
		t.Fatalf("Values = %#v, %#v, %d, %v", item, array, named, err)
	}
	generic, err := Generic[record](want)
	if generic != (record{}) || err != want {
		t.Fatalf("Generic = %#v, %v", generic, err)
	}
	value, err := Named(want)
	if value != 0 || err != want {
		t.Fatalf("Named = %d, %v", value, err)
	}
	valueText, err := Nested(want)
	if valueText != "" || err != want {
		t.Fatalf("Nested = %q, %v", valueText, err)
	}
	dynamic, err = NilSuccess()
	if dynamic != nil || err != nil {
		t.Fatalf("NilSuccess = %v, %v", dynamic, err)
	}
}
