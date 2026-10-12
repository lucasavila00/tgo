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
	a := [2]int{}
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = step("I")
	if error_tgo2 != nil {
		return error_tgo2
	}
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = step("V")
	if error_tgo4 != nil {
		return error_tgo4
	}
	a[value_tgo1] = value_tgo3
	fmt.Print(a)
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
