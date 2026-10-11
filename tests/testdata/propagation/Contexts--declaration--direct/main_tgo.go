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
	var tgo5 error = tgo4
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = step("A")
	tgo4 = tgo2
	if tgo4 != tgo5 {
		return tgo4
	}
	a := tgo1
	var tgo6 int
	var tgo7 error
	tgo6, tgo7 = step("B")
	tgo4 = tgo7
	if tgo4 != tgo5 {
		return tgo4
	}
	b := tgo6
	fmt.Print(a, b)
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
