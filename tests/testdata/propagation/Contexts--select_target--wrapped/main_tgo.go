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
	ch := make(chan int, 1)
	ch <- 1
	a := [2]int{}
	var value_tgo1 chan int = ch
	var tgo2 int
	select {
	case tgo2 = <-value_tgo1:
		var value_tgo3 int
		var error_tgo4 error
		value_tgo3, error_tgo4 = step("I")
		if error_tgo4 != nil {
			return fmt.Errorf("step: %w", error_tgo4)
		}
		a[value_tgo3] = tgo2
		fmt.Print("b")
	}

	return nil
}

func main() {
	for _, failure := range []int{0, 1, 2} {
		visit = 0
		failAt = failure
		err := run()
		if err == nil {
			fmt.Print("S|")
		} else if errors.Unwrap(err) == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
