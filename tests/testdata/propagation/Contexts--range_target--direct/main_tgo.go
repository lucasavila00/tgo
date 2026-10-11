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

func run() (tgo4 error) {
	var tgo5 error = tgo4
	var tgo6 error = tgo5

	a := [2]int{}
	{
		var tgo1 int

		for tgo1 = range []int{1, 2} {
			var tgo2 int
			var tgo3 error
			tgo2, tgo3 = step("I")
			tgo5 = tgo3
			if tgo5 != tgo6 {
				return tgo5
			}
			a[tgo2] = tgo1
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
