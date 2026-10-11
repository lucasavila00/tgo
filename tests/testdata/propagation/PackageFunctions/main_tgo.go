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
	tgo4 := tgo3
	tgo5 := tgo4
	defer func() { trace = append(trace, "defer") }()
	tgo1, tgo2 := next(fail)
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	return nil
}

var wrapped = func(fail bool) (tgo8 error) {
	tgo9 := tgo8
	tgo10 := tgo9
	defer func() { trace = append(trace, "defer") }()
	tgo6, tgo7 := next(fail)
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
			tgo15 := tgo14
			tgo16 := tgo15
			defer func() { trace = append(trace, "defer") }()
			tgo12, tgo13 := next(fail)
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
		tgo20 := tgo19
		tgo21 := tgo20
		defer func() { trace = append(trace, "defer") }()
		tgo17, tgo18 := next(fail)
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
	tgo25 := tgo24
	tgo26 := tgo25
	defer func() { trace = append(trace, "init-defer") }()
	{
		tgo22, tgo23 := next(true)
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
