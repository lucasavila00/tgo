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
		var tgo4 int
		var tgo5 error
		tgo4, tgo5 = step("I")
		if tgo5 != nil {
			return tgo5
		}
		var tgo6 bool = 0 == 0
		for i := tgo4; ; {
			if tgo6 {
				tgo6 = 0 != 0
			} else {
				i++
			}
			var tgo3 int = i
			var tgo1 int
			var tgo2 error
			tgo1, tgo2 = step("C")
			if tgo2 != nil {
				return tgo2
			}
			if !(tgo3 < tgo1+1) {
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
