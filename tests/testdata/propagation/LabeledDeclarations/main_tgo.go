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

func shortDirect(fail bool) error {
	defer func() { trace = append(trace, "defer") }()
	visits := 0
	goto L
L:
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = next(fail)
	if error_tgo2 != nil {
		return error_tgo2
	}
	value := value_tgo1
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
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = next(fail)
	if error_tgo4 != nil {
		return fmt.Errorf("next: %w", error_tgo4)
	}
	value := value_tgo3
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
	var value_tgo5 int
	var error_tgo6 error
	value_tgo5, error_tgo6 = next(fail)
	if error_tgo6 != nil {
		return error_tgo6
	}
	var value = value_tgo5
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
	var value_tgo7 int
	var error_tgo8 error
	value_tgo7, error_tgo8 = next(fail)
	if error_tgo8 != nil {
		return fmt.Errorf("next: %w", error_tgo8)
	}
	var value = value_tgo7
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
	var value_tgo9 int
	var error_tgo10 error
	value_tgo9, error_tgo10 = next(fail)
	if error_tgo10 != nil {
		return error_tgo10
	}
	value := value_tgo9
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
