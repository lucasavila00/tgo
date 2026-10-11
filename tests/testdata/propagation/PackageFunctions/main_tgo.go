package main

import fmt_tgo9 "fmt"

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

var direct = func(fail bool) (tgo3 error) {
	var tgo4 error = tgo3
	defer func() { trace = append(trace, "defer") }()
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next(fail)
	if tgo2 != nil {
		tgo4 = tgo2
		return tgo4
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var wrapped = func(fail bool) (tgo7 error) {
	var tgo8 error = tgo7
	defer func() { trace = append(trace, "defer") }()
	var tgo5 int
	var tgo6 error
	tgo5, tgo6 = next(fail)
	if tgo6 != nil {
		tgo8 = tgo6
		tgo8 = fmt_tgo9.Errorf("%s: %w", "next", tgo8)
		return tgo8
	}
	value := tgo5
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var collection = map[string][]func(bool) error{
	"run": {
		func(fail bool) (tgo12 error) {
			var tgo13 error = tgo12
			defer func() { trace = append(trace, "defer") }()
			var tgo10 int
			var tgo11 error
			tgo10, tgo11 = next(fail)
			if tgo11 != nil {
				tgo13 = tgo11
				return tgo13
			}
			value := tgo10
			trace = append(trace, fmt.Sprint(value))
			return nil
		},
	},
}

var nested = func() func(bool) error {
	return func(fail bool) (tgo16 error) {
		var tgo17 error = tgo16
		defer func() { trace = append(trace, "defer") }()
		var tgo14 int
		var tgo15 error
		tgo14, tgo15 = next(fail)
		if tgo15 != nil {
			tgo17 = tgo15
			return tgo17
		}
		value := tgo14
		trace = append(trace, fmt.Sprint(value))
		return nil
	}
}()

var initialized = func() (tgo20 error) {
	var tgo21 error = tgo20
	defer func() { trace = append(trace, "init-defer") }()
	{
		var tgo18 int
		var tgo19 error
		tgo18, tgo19 = next(true)
		if tgo19 != nil {
			tgo21 = tgo19
			return tgo21
		}
		_ = tgo18
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
