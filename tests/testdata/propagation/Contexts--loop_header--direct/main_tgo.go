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
		var value_tgo4 int
		var error_tgo5 error
		value_tgo4, error_tgo5 = step("I")
		if error_tgo5 != nil {
			return error_tgo5
		}
		var tgo6 bool = 0 == 0
		for i := value_tgo4; ; {
			if tgo6 {
				tgo6 = 0 != 0
			} else {
				i++
			}
			var value_tgo3 int = i
			var value_tgo1 int
			var error_tgo2 error
			value_tgo1, error_tgo2 = step("C")
			if error_tgo2 != nil {
				return error_tgo2
			}
			if !(value_tgo3 < value_tgo1+1) {
				break
			}
			{
				fmt.Print("b")
			}
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
		} else if err == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
