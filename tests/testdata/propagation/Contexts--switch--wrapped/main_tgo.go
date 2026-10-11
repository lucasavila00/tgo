package main

import tgo6 "fmt"

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

func run() (tgo3 error) {
	var tgo4 error = tgo3
	var tgo5 error = tgo4
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = step("T")
		tgo4 = tgo2
		if tgo4 != tgo5 {
			tgo4 = tgo6.Errorf("%s: %w", "step", tgo4)
			return tgo4
		}
		var tgo7 int = tgo1
		var tgo8 int = -1
		if tgo8 == -1 {
			var tgo9 int
			var tgo10 error
			tgo9, tgo10 = step("C")
			tgo4 = tgo10
			if tgo4 != tgo5 {
				tgo4 = tgo6.Errorf("%s: %w", "step", tgo4)
				return tgo4
			}
			if tgo7 == tgo9 {
				tgo8 = 0
			}
		}
		if tgo8 == -1 {
			if tgo7 ==

				2 {
				tgo8 = 1
			}
		}
		switch tgo8 {
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
