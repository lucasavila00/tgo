package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visit, failAt int

func step(name string) (int, error) {
	fmt.Print(name)
	visit++
	if visit == failAt {
		return 0, sentinel
	}
	return 1, nil
}

func predicate(name string) (bool, error) {
	_, err := step(name)
	return true, err
}

func pair() (int, string, error) {
	v, err := step("P")
	return v, "x", err
}

func only() error {
	_, err := step("O")
	return err
}

func identity[T any](v T) (T, error) {
	_, err := step("F")
	return v, err
}

func run() error {
	var tgo3 bool
	tgo3 = false
	if tgo3 {
		var value_tgo1 bool
		var error_tgo2 error
		value_tgo1, error_tgo2 = predicate("B")
		if error_tgo2 != nil {
			return error_tgo2
		}
		tgo3 = value_tgo1
	}
	fmt.Print(tgo3)
	var tgo6 bool
	tgo6 = true
	if tgo6 {
		var value_tgo4 bool
		var error_tgo5 error
		value_tgo4, error_tgo5 = predicate("C")
		if error_tgo5 != nil {
			return error_tgo5
		}
		tgo6 = value_tgo4
	}
	fmt.Print(tgo6)
	return nil
}

func main() {
	for _, failure := range []int{0, 1, 2} {
		visit = 0
		failAt = failure
		err := run()
		if err == nil {
			fmt.Print("S|")
		} else if err == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
