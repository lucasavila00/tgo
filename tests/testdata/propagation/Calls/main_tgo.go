package main

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
	var value_tgo3 int = effect("L")
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = target(fail)
	if error_tgo2 != nil {
		return error_tgo2
	}
	consume(value_tgo3, value_tgo1, effect("R"))
	return nil
}

func wrapped() error {
	var error_tgo4 error
	_, error_tgo4 = target(true)
	if error_tgo4 != nil {
		return fmt.Errorf("target: %w", error_tgo4)
	}
	return nil
}

func main() {
	fmt.Print(run(false) == nil, "|")
	fmt.Print(run(true) == sentinel, "|")
	e := wrapped()
	fmt.Print(e.Error(), "|", errors.Is(e, sentinel))
}
