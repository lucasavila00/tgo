package main

import fmt_tgo3 "fmt"

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
			return fmt_tgo3.Errorf("step: %w", tgo2)
		}
		var tgo4 int = tgo1
		var tgo5 int = -1
		if tgo5 == -1 {
			var tgo6 int
			var tgo7 error
			tgo6, tgo7 = step("C")
			if tgo7 != nil {
				return fmt_tgo3.Errorf("step: %w", tgo7)
			}
			if tgo4 == tgo6 {
				tgo5 = 0
			}
		}
		if tgo5 == -1 {
			if tgo4 ==

				2 {
				tgo5 = 1
			}
		}
		switch tgo5 {
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
