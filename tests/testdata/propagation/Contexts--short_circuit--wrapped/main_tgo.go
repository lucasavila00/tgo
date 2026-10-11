package main

import fmt_tgo5 "fmt"

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
	{
		var tgo6 bool
		tgo6 = false
		if tgo6 {
			var tgo1 bool
			var tgo2 error
			tgo1, tgo2 = predicate("B")
			if tgo2 != nil {
				tgo4 = tgo2
				tgo4 = fmt_tgo5.Errorf("%s: %w", "predicate", tgo4)
				return tgo4
			}
			tgo6 = tgo1
		}
		fmt.Print(tgo6)
	}
	{
		var tgo9 bool
		tgo9 = true
		if tgo9 {
			var tgo7 bool
			var tgo8 error
			tgo7, tgo8 = predicate("C")
			if tgo8 != nil {
				tgo4 = tgo8
				tgo4 = fmt_tgo5.Errorf("%s: %w", "predicate", tgo4)
				return tgo4
			}
			tgo9 = tgo7
		}
		fmt.Print(tgo9)
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
