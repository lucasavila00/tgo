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

func unnamed(label string) (tgo3 Value, tgo5 error) {
	var tgo4 Value = tgo3
	var tgo6 error = tgo5
	{
		var tgo1 Value
		var tgo2 error
		tgo1, tgo2 = get()
		if tgo2 != nil {
			tgo6 = tgo2
			return tgo4, tgo6
		}
		return tgo1, nil
	}
}

func named() (value Value, err error) {
	var tgo9 Value = value
	var tgo10 error = err
	value = 5
	err = sentinel
	{
		var tgo7 Value
		var tgo8 error
		tgo7, tgo8 = get()
		if tgo8 != nil {
			tgo10 = tgo8
			return tgo9, tgo10
		}
		_ = tgo7
	}
	return
}

func generic[T any](value T) (tgo13 T, tgo15 error) {
	var tgo14 T = tgo13
	var tgo16 error = tgo15
	{
		var tgo11 Value
		var tgo12 error
		tgo11, tgo12 = get()
		if tgo12 != nil {
			tgo16 = tgo12
			return tgo14, tgo16
		}
		_ = tgo11
	}
	return value, nil
}

func closure() error {
	call := func() (tgo19 Value, tgo21 error) {
		var tgo20 Value = tgo19
		var tgo22 error = tgo21
		{
			var tgo17 Value
			var tgo18 error
			tgo17, tgo18 = get()
			if tgo18 != nil {
				tgo22 = tgo18
				return tgo20, tgo22
			}
			return tgo17, nil
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
