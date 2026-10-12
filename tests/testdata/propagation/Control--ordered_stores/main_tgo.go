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

func run() error {
	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	a := [1]int{}
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = step("I")
	if error_tgo2 != nil {
		return error_tgo2
	}
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = step("V")
	if error_tgo4 != nil {
		return error_tgo4
	}
	var value_tgo5 int
	var error_tgo6 error
	value_tgo5, error_tgo6 = step("W")
	if error_tgo6 != nil {
		return error_tgo6
	}
	a[value_tgo1], a[0] = value_tgo3, value_tgo5
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
