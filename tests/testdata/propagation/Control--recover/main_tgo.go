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
	{
		var tgo1 int
		var tgo2 error
		tgo1, tgo2 = step("P")
		if tgo2 != nil {
			tgo4 = tgo2
			return tgo4
		}
		panic(tgo1)
	}

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
