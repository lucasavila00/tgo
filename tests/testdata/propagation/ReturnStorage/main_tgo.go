package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")

func next(fail bool) (int, error) {
	if fail {
		return 0, sentinel
	}
	return 7, nil
}

func named(fail, wrap bool) (value int, err error) {
	var zero_tgo3 int
	defer func() {
		value += 100
		if err != nil && !errors.Is(err, sentinel) {
			panic("defer received the wrong error")
		}
	}()
	value = 9
	if wrap {
		var value_tgo1 int
		var error_tgo2 error
		value_tgo1, error_tgo2 = next(fail)
		if error_tgo2 != nil {
			return zero_tgo3, fmt.Errorf("next: %w", error_tgo2)
		}
		value = value_tgo1
	} else {
		var value_tgo4 int
		var error_tgo5 error
		value_tgo4, error_tgo5 = next(fail)
		if error_tgo5 != nil {
			return zero_tgo3, error_tgo5
		}
		value = value_tgo4
	}
	return
}

func grouped(fail bool) (first, last error) {
	var zero_tgo7 error
	first = sentinel
	var error_tgo6 error
	_, error_tgo6 = next(fail)
	if error_tgo6 != nil {
		return zero_tgo7, error_tgo6
	}
	return
}

func blank(fail bool) (_, _ error) {
	var zero_tgo9 error
	var error_tgo8 error
	_, error_tgo8 = next(fail)
	if error_tgo8 != nil {
		return zero_tgo9, error_tgo8
	}
	return sentinel, nil
}

func main() {
	for _, fail := range []bool{false, true} {
		for _, wrap := range []bool{false, true} {
			value, err := named(fail, wrap)
			fmt.Println("named", fail, wrap, value, errors.Is(err, sentinel), err == sentinel)
		}
		first, last := grouped(fail)
		fmt.Println("grouped", fail, first == nil, last == sentinel)
		first, last = blank(fail)
		fmt.Println("blank", fail, first == nil, last == sentinel)
	}
}
