package main

import tgo4 "fmt"

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
	var tgo3 int
	defer func() {
		value += 100
		if err != nil && !errors.Is(err, sentinel) {
			panic("defer received the wrong error")
		}
	}()
	value = 9
	if wrap {
		{
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = next(fail)
			if tgo2 != nil {
				return tgo3, tgo4.Errorf("%s: %w", "next", tgo2)
			}
			value = tgo1
		}
	} else {
		{
			var tgo5 int
			var tgo6 error
			tgo5, tgo6 = next(fail)
			if tgo6 != nil {
				return tgo3, tgo6
			}
			value = tgo5
		}
	}
	return
}

func grouped(fail bool) (first, last error) {
	var tgo9 error
	first = sentinel
	{
		var tgo7 int
		var tgo8 error
		tgo7, tgo8 = next(fail)
		if tgo8 != nil {
			return tgo9, tgo8
		}
		_ = tgo7
	}
	return
}

func blank(fail bool) (_, _ error) {
	var tgo12 error
	{
		var tgo10 int
		var tgo11 error
		tgo10, tgo11 = next(fail)
		if tgo11 != nil {
			return tgo12, tgo11
		}
		_ = tgo10
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
