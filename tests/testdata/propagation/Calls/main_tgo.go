package main

import tgo12 "fmt"

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")

func effect(name string) int {
	fmt.Print(name)
	return 1
}

func target(fail bool) (int, error) {
	fmt.Print("T")
	if fail {
		return 0, sentinel
	}
	return 2, nil
}

func consume(a, b, c int) {
	fmt.Print(a, b, c)
}

func run(fail bool) (tgo3 error) {
	var tgo4 error = tgo3
	var tgo5 error = tgo4
	{
		var tgo6 int = effect("L")
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = target(fail)
		tgo4 = tgo2
		if tgo4 != tgo5 {
			return tgo4
		}
		consume(tgo6, tgo1, effect("R"))
	}
	return nil
}

func wrapped() (tgo9 error) {
	var tgo10 error = tgo9
	var tgo11 error = tgo10
	{
		var tgo7 int
		var tgo8 error
		tgo7, tgo8 = target(true)
		tgo10 = tgo8
		if tgo10 != tgo11 {
			tgo10 = tgo12.Errorf("%s: %w", "target", tgo10)
			return tgo10
		}
		_ = tgo7
	}
	return nil
}

func main() {
	fmt.Print(run(false) == nil, "|")
	fmt.Print(run(true) == sentinel, "|")
	e := wrapped()
	fmt.Print(e.Error(), "|", errors.Is(e, sentinel))
}
