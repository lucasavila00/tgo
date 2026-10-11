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

func run() (tgo3 error) {
	var tgo4 error = tgo3
	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	a := [1]int{}
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = step("I")
		if tgo2 != nil {
			tgo4 = tgo2
			return tgo4
		}
		var tgo5 int
		var tgo6 error
		tgo5, tgo6 = step("V")
		if tgo6 != nil {
			tgo4 = tgo6
			return tgo4
		}
		var tgo7 int
		var tgo8 error
		tgo7, tgo8 = step("W")
		if tgo8 != nil {
			tgo4 = tgo8
			return tgo4
		}
		a[tgo1], a[0] = tgo5, tgo7
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
