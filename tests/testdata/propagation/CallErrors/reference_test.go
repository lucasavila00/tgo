package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func reference[E error](err E, failed, wrapped bool) (int, error) {
	defer done()
	value, callErr := next(err)
	if failed {
		if wrapped {
			return 0, fmt.Errorf("%s: %w", "next", callErr)
		}
		return 0, callErr
	}
	return use(value), nil
}

func check[E error](t *testing.T, err E, failed bool, direct, wrapped func(E) (int, error)) {
	t.Helper()
	for _, wrapping := range []bool{false, true} {
		trace = ""
		wantValue, wantError := reference(err, failed, wrapping)
		wantTrace := trace
		trace = ""
		run := direct
		if wrapping {
			run = wrapped
		}
		value, actual := run(err)
		if value != wantValue || trace != wantTrace {
			t.Fatalf("value/trace: %d %q, want %d %q", value, trace, wantValue, wantTrace)
		}
		if wantError == nil {
			if actual != nil {
				t.Fatalf("success returned %v", actual)
			}
			continue
		}
		if reflect.TypeOf(actual) != reflect.TypeOf(wantError) {
			t.Fatalf("error type: %T, want %T", actual, wantError)
		}
		if wrapping {
			if actual.Error() != wantError.Error() {
				t.Fatalf("wrapped error: %q, want %q", actual.Error(), wantError.Error())
			}
			actual = errors.Unwrap(actual)
		}
		if reflect.TypeOf(err).Comparable() && actual != error(err) {
			t.Fatalf("error identity: %v", actual)
		}
		// DeepEqual also checks error values that Go cannot compare.
		if !reflect.DeepEqual(actual, err) {
			t.Fatalf("error: %#v, want %#v", actual, err)
		}
	}
}

func TestCallErrors(t *testing.T) {
	var pointer *PointerError
	pointerValues := []*PointerError{pointer, {}}
	for _, err := range pointerValues {
		check(t, err, err != nil, pointerDirect, pointerWrapped)
		check(t, err, err != nil, genericDirect[*PointerError], genericWrapped[*PointerError])
	}
	for _, err := range []error{nil, pointer, &PointerError{}, NumberError(0), StructError{}} {
		check(t, err, err != nil, ifaceDirect, ifaceWrapped)
		check(t, err, err != nil, genericDirect[error], genericWrapped[error])
	}
	for _, err := range []NumberError{0, 1} {
		check(t, err, true, numberDirect, numberWrapped)
		check(t, err, true, genericDirect[NumberError], genericWrapped[NumberError])
		trace = ""
		value, actual := comparableResult(err)
		if value != 0 || actual != err || trace != "ND" {
			t.Fatalf("comparable zero error: %d %v %q", value, actual, trace)
		}
	}
	for _, err := range []SliceError{nil, {}, {1}} {
		check(t, err, err != nil, sliceDirect, sliceWrapped)
		check(t, err, err != nil, genericDirect[SliceError], genericWrapped[SliceError])
		trace = ""
		value, actual := shadow(0, err)
		wantValue, wantTrace := 0, "ND"
		if err == nil {
			wantValue, wantTrace = 7, "NUD"
		}
		if value != wantValue || trace != wantTrace || (actual == nil) != (err == nil) {
			t.Fatalf("nil binding: %d %v %q", value, actual, trace)
		}
	}
	for _, err := range []MapError{nil, {}, {1: 2}} {
		check(t, err, err != nil, mappedDirect, mappedWrapped)
		check(t, err, err != nil, genericDirect[MapError], genericWrapped[MapError])
	}
	for _, err := range []ChanError{nil, make(ChanError)} {
		check(t, err, err != nil, channelDirect, channelWrapped)
		check(t, err, err != nil, genericDirect[ChanError], genericWrapped[ChanError])
	}
	// Non-nil functions need pointer checks because DeepEqual does not compare them.
	check(t, FuncError(nil), false, functionDirect, functionWrapped)
	trace = ""
	_, actual := functionDirect(func() {})
	if actual == nil || trace != "ND" {
		t.Fatalf("function error: %v %q", actual, trace)
	}
	check(t, StructError{}, true, structureDirect, structureWrapped)
	check(t, StructError{Values: []int{1}}, true, genericDirect[StructError], genericWrapped[StructError])
}

func TestGenericReturn(t *testing.T) {
	var pointer *PointerError
	trace = ""
	value, err := genericReturn(pointer)
	if value != 7 || err != nil || trace != "NUD" {
		t.Fatalf("pointer success: %d %v %q", value, err, trace)
	}
	var iface error = pointer
	trace = ""
	value, actual := genericReturn(iface)
	if value != 0 || actual != iface || trace != "ND" {
		t.Fatalf("interface failure: %d %v %q", value, actual, trace)
	}
	trace = ""
	value, zero := genericReturn(NumberError(0))
	if value != 0 || zero != 0 || trace != "ND" {
		t.Fatalf("zero failure: %d %v %q", value, zero, trace)
	}
}

func TestPointerContexts(t *testing.T) {
	cases := []struct {
		name            string
		direct, wrapped func(*PointerError) (int, error)
		prefix          string
		skipped         bool
	}{
		{"assign", contextassignDirect, contextassignWrapped, "N", false},
		{"return", contextreturnDirect, contextreturnWrapped, "N", false},
		{"loopInit", contextloopInitDirect, contextloopInitWrapped, "N", false},
		{"loopPost", contextloopPostDirect, contextloopPostWrapped, "UN", false},
		{"switch", contextswitchDirect, contextswitchWrapped, "N", false},
		{"caseExpr", contextcaseExprDirect, contextcaseExprWrapped, "N", false},
		{"shortCircuit", contextshortCircuitDirect, contextshortCircuitWrapped, "N", true},
		{"literal", contextliteralDirect, contextliteralWrapped, "N", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, err := range []*PointerError{nil, {}} {
				for _, wrapping := range []bool{false, true} {
					run := c.direct
					if wrapping {
						run = c.wrapped
					}
					trace = ""
					_, actual := run(err)
					if c.skipped {
						if actual != nil || trace != "UD" {
							t.Fatalf("skipped: %v %q", actual, trace)
						}
						continue
					}
					if err == nil {
						if actual != nil || trace != c.prefix+"UD" && !(c.name == "loopPost" && trace == "UND") {
							t.Fatalf("success: %v %q", actual, trace)
						}
						continue
					}
					if trace != c.prefix+"D" {
						t.Fatalf("failure trace: %q", trace)
					}
					if wrapping {
						if actual == nil || actual.Error() != "next: pointer" {
							t.Fatalf("wrap: %v", actual)
						}
						actual = errors.Unwrap(actual)
					}
					if actual != err {
						t.Fatalf("identity: %v", actual)
					}
				}
			}
		})
	}
}

func TestSharedErrorStorage(t *testing.T) {
	slice := SliceError{1}
	_, actual := genericDirect(slice)
	actual.(SliceError)[0] = 2
	if slice[0] != 2 {
		t.Fatal("slice error storage changed")
	}
	mapped := MapError{1: 2}
	_, actual = mappedWrapped(mapped)
	errors.Unwrap(actual).(MapError)[1] = 3
	if mapped[1] != 3 {
		t.Fatal("map error storage changed")
	}
	function := FuncError(func() {})
	for _, run := range []func(FuncError) (int, error){functionDirect, functionWrapped, genericDirect[FuncError], genericWrapped[FuncError]} {
		_, actual = run(function)
		if unwrapped := errors.Unwrap(actual); unwrapped != nil {
			actual = unwrapped
		}
		if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(function).Pointer() {
			t.Fatal("function error changed")
		}
	}
}

func TestConstrainedErrors(t *testing.T) {
	for _, err := range []*PointerError{nil, {}} {
		check(t, err, err != nil, constrainedPointerDirect[*PointerError], constrainedPointerWrapped[*PointerError])
		check(t, err, err != nil, constrainedMixedDirect[*PointerError], constrainedMixedWrapped[*PointerError])
	}
	for _, err := range []NumberError{0, 1} {
		check(t, err, true, constrainedMixedDirect[NumberError], constrainedMixedWrapped[NumberError])
	}
	for _, err := range []SliceError{nil, {}, {1}} {
		check(t, err, err != nil, constrainedNilableDirect[SliceError], constrainedNilableWrapped[SliceError])
	}
	for _, err := range []MapError{nil, {}, {1: 2}} {
		check(t, err, err != nil, constrainedNilableDirect[MapError], constrainedNilableWrapped[MapError])
	}
}

func TestPointerNilShadow(t *testing.T) {
	for _, err := range []*PointerError{nil, {}} {
		trace = ""
		value, actual := shadowPointer(0, err)
		if err == nil {
			if value != 7 || actual != nil || trace != "NUD" {
				t.Fatalf("nil pointer: %d %v %q", value, actual, trace)
			}
		} else if value != 0 || actual != err || trace != "ND" {
			t.Fatalf("pointer failure: %d %v %q", value, actual, trace)
		}
	}
}
