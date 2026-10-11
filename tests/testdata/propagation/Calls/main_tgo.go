package main

import fmt_tgo10 "fmt"

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
	{
		var tgo5 int = effect("L")
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = target(fail)
		if tgo2 != nil {
			tgo4 = tgo2
			return tgo4
		}
		consume(tgo5, tgo1, effect("R"))
	}
	return nil
}

func wrapped() (tgo8 error) {
	var tgo9 error = tgo8
	{
		var tgo6 int
		var tgo7 error
		tgo6, tgo7 = target(true)
		if tgo7 != nil {
			tgo9 = tgo7
			tgo9 = fmt_tgo10.Errorf("%s: %w", "target", tgo9)
			return tgo9
		}
		_ = tgo6
	}
	return nil
}

func main() {
	fmt.Print(run(false) == nil, "|")
	fmt.Print(run(true) == sentinel, "|")
	e := wrapped()
	fmt.Print(e.Error(), "|", errors.Is(e, sentinel))
}
