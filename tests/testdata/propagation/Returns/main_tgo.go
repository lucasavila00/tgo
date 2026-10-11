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
	var tgo3 Value
	{
		var tgo1 Value
		var tgo2 error
		tgo1, tgo2 = get()
		if tgo2 != nil {
			return tgo3, tgo2
		}
		return tgo1, nil
	}
}

func named() (value Value, err error) {
	var tgo6 Value = value
	value = 5
	err = sentinel
	{
		var tgo4 Value
		var tgo5 error
		tgo4, tgo5 = get()
		if tgo5 != nil {
			return tgo6, tgo5
		}
		_ = tgo4
	}
	return
}

func generic[T any](value T) (T, error) {
	var tgo9 T
	{
		var tgo7 Value
		var tgo8 error
		tgo7, tgo8 = get()
		if tgo8 != nil {
			return tgo9, tgo8
		}
		_ = tgo7
	}
	return value, nil
}

func closure() error {
	call := func() (Value, error) {
		var tgo12 Value
		{
			var tgo10 Value
			var tgo11 error
			tgo10, tgo11 = get()
			if tgo11 != nil {
				return tgo12, tgo11
			}
			return tgo10, nil
		}
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
