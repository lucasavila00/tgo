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
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = step("T")
		if tgo2 != nil {
			return tgo2
		}
		var tgo3 int = tgo1
		var tgo4 int = -1
		if tgo4 == -1 {
			var tgo5 int
			var tgo6 error
			tgo5, tgo6 = step("C")
			if tgo6 != nil {
				return tgo6
			}
			if tgo3 == tgo5 {
				tgo4 = 0
			}
		}
		if tgo4 == -1 {
			if tgo3 ==

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
		} else if err == sentinel {
			fmt.Print("E|")
		} else {
			panic(err)
		}
	}
}
