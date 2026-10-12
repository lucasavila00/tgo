package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var fail bool

type Value int

func get() (Value, error) {
	if fail {
		return 7, sentinel
	}
	return 8, nil
}

func unnamed(label string) (Value, error) {
	var zero_tgo3 Value
	var value_tgo1 Value
	var error_tgo2 error
	value_tgo1, error_tgo2 = get()
	if error_tgo2 != nil {
		return zero_tgo3, error_tgo2
	}
	return value_tgo1, nil
}

func named() (value Value, err error) {
	var zero_tgo5 Value
	value = 5
	err = sentinel
	var error_tgo4 error
	_, error_tgo4 = get()
	if error_tgo4 != nil {
		return zero_tgo5, error_tgo4
	}
	return
}

func generic[T any](value T) (T, error) {
	var zero_tgo7 T
	var error_tgo6 error
	_, error_tgo6 = get()
	if error_tgo6 != nil {
		return zero_tgo7, error_tgo6
	}
	return value, nil
}

func closure() error {
	call := func() (Value, error) {
		var zero_tgo10 Value
		var value_tgo8 Value
		var error_tgo9 error
		value_tgo8, error_tgo9 = get()
		if error_tgo9 != nil {
			return zero_tgo10, error_tgo9
		}
		return value_tgo8, nil
	}
	v, e := call()
	fmt.Print(v, e == sentinel)
	return nil
}

func main() {
	for _, failure := range []bool{false, true} {
		fail = failure
		v, e := unnamed("shadow")
		fmt.Print(v, e == sentinel, "|")
		v, e = named()
		fmt.Print(v, e == sentinel, "|")
		s, e := generic("value")
		fmt.Print(s, e == sentinel, "|")
		closure()
		fmt.Print("|")
	}
}
