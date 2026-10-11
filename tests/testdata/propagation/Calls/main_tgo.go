package main

import fmt_tgo6 "fmt"

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

func run(fail bool) error {
	{
		var tgo3 int = effect("L")
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = target(fail)
		if tgo2 != nil {
			return tgo2
		}
		consume(tgo3, tgo1, effect("R"))
	}
	return nil
}

func wrapped() error {
	{
		var tgo4 int
		var tgo5 error
		tgo4, tgo5 = target(true)
		if tgo5 != nil {
			return fmt_tgo6.Errorf("target: %w", tgo5)
		}
		_ = tgo4
	}
	return nil
}

func main() {
	fmt.Print(run(false) == nil, "|")
	fmt.Print(run(true) == sentinel, "|")
	e := wrapped()
	fmt.Print(e.Error(), "|", errors.Is(e, sentinel))
}
