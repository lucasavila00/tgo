package successreturn

import (
	"errors"
	"strings"
	"testing"
)

func TestSuccessfulReturns(t *testing.T) {
	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "one", got: first(One()), want: 1},
		{name: "comment", got: first(Commented()), want: "comment"},
		{name: "generic", got: first(Generic("generic")), want: "generic"},
		{name: "named", got: first(Named(3)), want: 3},
		{name: "method", got: first(Loader{Value: 4}.Load()), want: 4},
		{name: "literal", got: first(FromLiteral(5)), want: 5},
		{name: "wrapped", got: first(Wrapped(true)), want: 7},
		{name: "transparent", got: first(Transparent(true)), want: 7},
		{name: "only success return", got: first(OnlySuccessReturn()), want: 9},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s = %v, want %v", test.name, test.got, test.want)
		}
	}
	left, right, err := Many()
	if left != 2 || right != "two" || err != nil {
		t.Fatalf("Many = %d, %q, %v", left, right, err)
	}
}

func TestPropagationFailure(t *testing.T) {
	_, wrapped := Wrapped(false)
	if !errors.Is(wrapped, ErrFailure) || !strings.Contains(wrapped.Error(), "load: ") {
		t.Fatalf("Wrapped error = %v", wrapped)
	}
	_, transparent := Transparent(false)
	if transparent != ErrFailure {
		t.Fatalf("Transparent error = %v", transparent)
	}
}

func first[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
