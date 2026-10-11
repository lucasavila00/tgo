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
	var tgo7 error = tgo6
	{
		var tgo1 Value
		var tgo2 error
		tgo1, tgo2 = get()
		tgo6 = tgo2
		if tgo6 != tgo7 {
			return tgo4, tgo6
		}
		return tgo1, nil
	}
}

func named() (value Value, err error) {
	var tgo10 Value = value
	var tgo11 error = err
	var tgo12 error = tgo11

	value = 5
	err = sentinel
	{
		var tgo8 Value
		var tgo9 error
		tgo8, tgo9 = get()
		tgo11 = tgo9
		if tgo11 != tgo12 {
			return tgo10, tgo11
		}
		_ = tgo8
	}
	return
}

func generic[T any](value T) (tgo15 T, tgo17 error) {
	var tgo16 T = tgo15
	var tgo18 error = tgo17
	var tgo19 error = tgo18
	{
		var tgo13 Value
		var tgo14 error
		tgo13, tgo14 = get()
		tgo18 = tgo14
		if tgo18 != tgo19 {
			return tgo16, tgo18
		}
		_ = tgo13
	}
	return value, nil
}

func closure() error {
	call := func() (tgo22 Value, tgo24 error) {
		var tgo23 Value = tgo22
		var tgo25 error = tgo24
		var tgo26 error = tgo25
		{
			var tgo20 Value
			var tgo21 error
			tgo20, tgo21 = get()
			tgo25 = tgo21
			if tgo25 != tgo26 {
				return tgo23, tgo25
			}
			return tgo20, nil
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
