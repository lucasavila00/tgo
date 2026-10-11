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

func run() (tgo3 error) {
	var tgo4 error = tgo3
	{
		var tgo5 bool
		tgo5 = false
		if tgo5 {
			var tgo1 bool
			var tgo2 error
			tgo1, tgo2 = predicate("B")
			if tgo2 != nil {
				tgo4 = tgo2
				return tgo4
			}
			tgo5 = tgo1
		}
		fmt.Print(tgo5)
	}
	{
		var tgo8 bool
		tgo8 = true
		if tgo8 {
			var tgo6 bool
			var tgo7 error
			tgo6, tgo7 = predicate("C")
			if tgo7 != nil {
				tgo4 = tgo7
				return tgo4
			}
			tgo8 = tgo6
		}
		fmt.Print(tgo8)
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
