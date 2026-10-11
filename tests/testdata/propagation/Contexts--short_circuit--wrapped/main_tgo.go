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
		var tgo4 bool
		tgo4 = false
		if tgo4 {
			var tgo1 bool
			var tgo2 error
			tgo1, tgo2 = predicate("B")
			if tgo2 != nil {
				return fmt_tgo3.Errorf("predicate: %w", tgo2)
			}
			tgo4 = tgo1
		}
		fmt.Print(tgo4)
	}
	{
		var tgo7 bool
		tgo7 = true
		if tgo7 {
			var tgo5 bool
			var tgo6 error
			tgo5, tgo6 = predicate("C")
			if tgo6 != nil {
				return fmt_tgo3.Errorf("predicate: %w", tgo6)
			}
			tgo7 = tgo5
		}
		fmt.Print(tgo7)
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
