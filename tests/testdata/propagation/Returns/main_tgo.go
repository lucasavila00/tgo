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

func unnamed(Value string) (tgo3 Value, tgo5 error) {
	tgo4 := tgo3
	tgo6 := tgo5
	tgo7 := tgo6
	{
		tgo1, tgo2 := get()
		tgo6 = tgo2
		if tgo6 != tgo7 {
			return tgo4, tgo6
		}
		return tgo1, nil
	}
}

func named() (value Value, err error) {
	tgo10 := value
	tgo11 := err
	tgo12 := tgo11
	value = 5
	err = sentinel
	{
		tgo8, tgo9 := get()
		tgo11 = tgo9
		if tgo11 != tgo12 {
			return tgo10, tgo11
		}
		_ = tgo8
	}
	return
}

func generic[T any](value T) (tgo15 T, tgo17 error) {
	tgo16 := tgo15
	tgo18 := tgo17
	tgo19 := tgo18
	{
		tgo13, tgo14 := get()
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
		tgo23 := tgo22
		tgo25 := tgo24
		tgo26 := tgo25
		{
			tgo20, tgo21 := get()
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
