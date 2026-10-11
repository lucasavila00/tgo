package main

import fmt_tgo5 "fmt"

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
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next(fail)
	if tgo2 != nil {
		return tgo2
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var wrapped = func(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	var tgo3 int
	var tgo4 error
	tgo3, tgo4 = next(fail)
	if tgo4 != nil {
		return fmt_tgo5.Errorf("%s: %w", "next", tgo4)
	}
	value := tgo3
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var collection = map[string][]func(bool) error{
	"run": {
		func(fail bool) error {
			defer func() { trace = append(trace, "defer") }()
			var tgo6 int
			var tgo7 error
			tgo6, tgo7 = next(fail)
			if tgo7 != nil {
				return tgo7
			}
			value := tgo6
			trace = append(trace, fmt.Sprint(value))
			return nil
		},
	},
}

var nested = func() func(bool) error {
	return func(fail bool) error {
		defer func() { trace = append(trace, "defer") }()
		var tgo8 int
		var tgo9 error
		tgo8, tgo9 = next(fail)
		if tgo9 != nil {
			return tgo9
		}
		value := tgo8
		trace = append(trace, fmt.Sprint(value))
		return nil
	}
}()

var initialized = func() error {
	defer func() { trace = append(trace, "init-defer") }()
	{
		var tgo10 int
		var tgo11 error
		tgo10, tgo11 = next(true)
		if tgo11 != nil {
			return tgo11
		}
		_ = tgo10
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
