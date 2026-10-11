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
	var tgo1 int
	var tgo2 error
	tgo1, tgo2 = step("A")
	if tgo2 != nil {
		return tgo2
	}
	var tgo5 S = ([]S{{tgo1}})[0]
	var tgo3 int
	var tgo4 error
	tgo3, tgo4 = step("B")
	if tgo4 != nil {
		return tgo4
	}
	a := []S{tgo5, {tgo3}}
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
