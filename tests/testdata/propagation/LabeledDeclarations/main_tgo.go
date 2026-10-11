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

func shortDirect(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = next(fail)
	if tgo2 != nil {
		return tgo2
	}
	value := tgo1
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func shortWrapped(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo3 int
	var tgo4 error
	tgo3, tgo4 = next(fail)
	if tgo4 != nil {
		return fmt_tgo5.Errorf("%s: %w", "next", tgo4)
	}
	value := tgo3
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func varDirect(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo6 int
	var tgo7 error
	tgo6, tgo7 = next(fail)
	if tgo7 != nil {
		return tgo7
	}
	var value = tgo6
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func varWrapped(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var tgo8 int
	var tgo9 error
	tgo8, tgo9 = next(fail)
	if tgo9 != nil {
		return fmt_tgo5.Errorf("%s: %w", "next", tgo9)
	}
	var value = tgo8
	trace = append(trace, fmt.Sprint(value))
	visits++
	if visits < 2 {
		goto L
	}
	return nil
}

func nestedLabels(fail bool) error {
	goto Outer
Outer:
Inner:
	var tgo10 int
	var tgo11 error
	tgo10, tgo11 = next(fail)
	if tgo11 != nil {
		return tgo11
	}
	value := tgo10
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
