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
		var tgo7 bool
		tgo7 = false
		if tgo7 {
			var tgo1 bool
			var tgo2 error
			tgo1, tgo2 = predicate("B")
			tgo4 = tgo2
			if tgo4 != tgo5 {
				tgo4 = tgo6.Errorf("%s: %w", "predicate", tgo4)
				return tgo4
			}
			tgo7 = tgo1
		}
		fmt.Print(tgo7)
	}
	{
		var tgo10 bool
		tgo10 = true
		if tgo10 {
			var tgo8 bool
			var tgo9 error
			tgo8, tgo9 = predicate("C")
			tgo4 = tgo9
			if tgo4 != tgo5 {
				tgo4 = tgo6.Errorf("%s: %w", "predicate", tgo4)
				return tgo4
			}
			tgo10 = tgo8
		}
		fmt.Print(tgo10)
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
