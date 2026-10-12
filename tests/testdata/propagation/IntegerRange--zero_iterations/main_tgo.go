package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("failure")
var visit, failAt int

type N uint64

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
	var a *[2]N
	var tgo1 N
	for tgo1 = range 0 {
		var value_tgo2 *[2]N = a
		var value_tgo3 int
		var error_tgo4 error
		value_tgo3, error_tgo4 = step("I")
		if error_tgo4 != nil {
			return error_tgo4
		}
		value_tgo2[value_tgo3] = tgo1
		{

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
