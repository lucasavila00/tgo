package main

import (
	"errors"
	"fmt"
	"reflect"
)

var sentinel = errors.New("sentinel")
var trace []string

func next(fail bool) (int, error) {
	trace = append(trace, "call")
	if fail {
		return 7, sentinel
	}
	return 7, nil
}

var direct = func(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = next(fail)
	if error_tgo2 != nil {
		return error_tgo2
	}
	value := value_tgo1
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var wrapped = func(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = next(fail)
	if error_tgo4 != nil {
		return fmt.Errorf("next: %w", error_tgo4)
	}
	value := value_tgo3
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var collection = map[string][]func(bool) error{
	"run": {
		func(fail bool) error {
			defer func() { trace = append(trace, "defer") }()
			var value_tgo5 int
			var error_tgo6 error
			value_tgo5, error_tgo6 = next(fail)
			if error_tgo6 != nil {
				return error_tgo6
			}
			value := value_tgo5
			trace = append(trace, fmt.Sprint(value))
			return nil
		},
	},
}

var nested = func() func(bool) error {
	return func(fail bool) error {
		defer func() { trace = append(trace, "defer") }()
		var value_tgo7 int
		var error_tgo8 error
		value_tgo7, error_tgo8 = next(fail)
		if error_tgo8 != nil {
			return error_tgo8
		}
		value := value_tgo7
		trace = append(trace, fmt.Sprint(value))
		return nil
	}
}()

var initialized = func() error {
	defer func() { trace = append(trace, "init-defer") }()
	var error_tgo9 error
	_, error_tgo9 = next(true)
	if error_tgo9 != nil {
		return error_tgo9
	}
	trace = append(trace, "unreachable")
	return nil
}()

var after = func() int {
	trace = append(trace, "after-init")
	return 1
}()

func main() {
	if initialized != sentinel || after != 1 || !reflect.DeepEqual(trace, []string{"call", "init-defer", "after-init"}) {
		panic(fmt.Sprint("initialization: ", initialized, " ", trace))
	}
	for _, entry := range []struct {
		name string
		run  func(bool) error
		wrap bool
	}{
		{"direct", direct, false},
		{"wrapped", wrapped, true},
		{"collection", collection["run"][0], false},
		{"nested", nested, false},
	} {
		for _, fail := range []bool{false, true} {
			trace = nil
			got := entry.run(fail)
			actualTrace := append([]string(nil), trace...)
			trace = nil
			want := reference(fail, entry.wrap)
			if fmt.Sprint(got) != fmt.Sprint(want) || !reflect.DeepEqual(actualTrace, trace) || errors.Is(got, sentinel) != errors.Is(want, sentinel) {
				panic(fmt.Sprint(entry.name, ": ", got, " ", actualTrace))
			}
			if fail && !entry.wrap && got != sentinel {
				panic("error identity changed")
			}
			fmt.Println(entry.name, fail, got, actualTrace)
		}
	}
}
