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
	type S struct{ v int }
	var value_tgo1 int
	var error_tgo2 error
	value_tgo1, error_tgo2 = step("A")
	if error_tgo2 != nil {
		return error_tgo2
	}
	var value_tgo5 S = ([]S{{value_tgo1}})[0]
	var value_tgo3 int
	var error_tgo4 error
	value_tgo3, error_tgo4 = step("B")
	if error_tgo4 != nil {
		return error_tgo4
	}
	a := []S{value_tgo5, {value_tgo3}}
	fmt.Print(a)
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
