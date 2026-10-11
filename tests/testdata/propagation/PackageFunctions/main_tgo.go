package main

import tgo11 "fmt"

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
	var tgo5 error = tgo4

	defer func() { trace = append(trace, "defer") }()
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next(fail)
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var wrapped = func(fail bool) (tgo8 error) {
	var tgo9 error = tgo8
	var tgo10 error = tgo9

	defer func() { trace = append(trace, "defer") }()
	var tgo6 int
	var tgo7 error
	tgo6, tgo7 = next(fail)
	tgo9 = tgo7
	if tgo9 != tgo10 {
		tgo9 = tgo11.Errorf("%s: %w", "next", tgo9)
		return tgo9
	}
	value := tgo6
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var collection = map[string][]func(bool) error{
	"run": {
		func(fail bool) (tgo14 error) {
			var tgo15 error = tgo14
			var tgo16 error = tgo15

			defer func() { trace = append(trace, "defer") }()
			var tgo12 int
			var tgo13 error
			tgo12, tgo13 = next(fail)
			tgo15 = tgo13
			if tgo15 != tgo16 {
				return tgo15
			}
			value := tgo12
			trace = append(trace, fmt.Sprint(value))
			return nil
		},
	},
}

var nested = func() func(bool) error {
	return func(fail bool) (tgo19 error) {
		var tgo20 error = tgo19
		var tgo21 error = tgo20

		defer func() { trace = append(trace, "defer") }()
		var tgo17 int
		var tgo18 error
		tgo17, tgo18 = next(fail)
		tgo20 = tgo18
		if tgo20 != tgo21 {
			return tgo20
		}
		value := tgo17
		trace = append(trace, fmt.Sprint(value))
		return nil
	}
}()

var initialized = func() (tgo24 error) {
	var tgo25 error = tgo24
	var tgo26 error = tgo25

	defer func() { trace = append(trace, "init-defer") }()
	{
		var tgo22 int
		var tgo23 error
		tgo22, tgo23 = next(true)
		tgo25 = tgo23
		if tgo25 != tgo26 {
			return tgo25
		}
		_ = tgo22
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
