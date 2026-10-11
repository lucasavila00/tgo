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
	tgo4 := tgo3
	tgo5 := tgo4
	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	{
		tgo1, tgo2 := step("P")
		tgo4 = tgo2
		if tgo4 != tgo5 {
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
