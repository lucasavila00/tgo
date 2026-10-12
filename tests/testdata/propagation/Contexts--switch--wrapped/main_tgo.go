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
	{
		var value_tgo1 int
		var error_tgo2 error
		value_tgo1, error_tgo2 = step("T")
		if error_tgo2 != nil {
			return fmt.Errorf("step: %w", error_tgo2)
		}
		var value_tgo3 int = value_tgo1
		var tgo4 int = -1
		if tgo4 == -1 {
			var value_tgo5 int
			var error_tgo6 error
			value_tgo5, error_tgo6 = step("C")
			if error_tgo6 != nil {
				return fmt.Errorf("step: %w", error_tgo6)
			}
			if value_tgo3 == value_tgo5 {
				tgo4 = 0
			}
		}
		if tgo4 == -1 {
			if value_tgo3 ==

				2 {
				tgo4 = 1
			}
		}
		switch tgo4 {
		case 0:
			fmt.Print("m")
			fallthrough
		case 1:
			fmt.Print("f")
		}
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
