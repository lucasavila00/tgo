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

func shortDirect(fail bool) (tgo3 error) {
	var tgo4 error = tgo3
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next(fail)
	if tgo2 != nil {
		tgo4 = tgo2
		return tgo4
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func shortWrapped(fail bool) (tgo7 error) {
	var tgo8 error = tgo7
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
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
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func varDirect(fail bool) (tgo12 error) {
	var tgo13 error = tgo12
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo10 int
	var tgo11 error
	tgo10, tgo11 = next(fail)
	if tgo11 != nil {
		tgo13 = tgo11
		return tgo13
	}
	var value = tgo10
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func varWrapped(fail bool) (tgo16 error) {
	var tgo17 error = tgo16
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo14 int
	var tgo15 error
	tgo14, tgo15 = next(fail)
	if tgo15 != nil {
		tgo17 = tgo15
		tgo17 = fmt_tgo9.Errorf("%s: %w", "next", tgo17)
		return tgo17
	}
	var value = tgo14
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func nestedLabels(fail bool) (tgo20 error) {
	var tgo21 error = tgo20
	goto Outer
Outer:
Inner:
	var tgo18 int
	var tgo19 error
	tgo18, tgo19 = next(fail)
	if tgo19 != nil {
		tgo21 = tgo19
		return tgo21
	}
	value := tgo18
	trace = append(trace, fmt.Sprint(value))
	if false {
		goto Inner
	}
	return nil
}

func main() {
	for _, entry := range []struct {
		name   string
		run    func(bool) error
		wrap   bool
		visits int
	}{
		{"short-direct", shortDirect, false, 2},
		{"short-wrapped", shortWrapped, true, 2},
		{"var-direct", varDirect, false, 2},
		{"var-wrapped", varWrapped, true, 2},
	} {
		for _, fail := range []bool{false, true} {
			trace = nil
			got := entry.run(fail)
			actual := append([]string(nil), trace...)
			trace = nil
			want := reference(fail, entry.wrap, entry.visits)
			if fmt.Sprint(got) != fmt.Sprint(want) || !reflect.DeepEqual(actual, trace) || errors.Is(got, sentinel) != errors.Is(want, sentinel) {
				panic(entry.name)
			}
			if fail && !entry.wrap && got != sentinel {
				panic("error identity changed")
			}
			fmt.Println(entry.name, fail, got, actual)
		}
	}
	trace = nil
	if nestedLabels(false) != nil || !reflect.DeepEqual(trace, []string{"call", "7"}) {
		panic("nested labels")
	}
	fmt.Println("nested labels", trace)
}
