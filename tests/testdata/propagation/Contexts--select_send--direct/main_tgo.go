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
	ch := make(chan int, 1)
	{
		var tgo1 chan int = ch
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = step("V")
		if tgo3 != nil {
			return tgo3
		}
		select {
		case tgo1 <- tgo2:
			fmt.Print("b")
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
