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

func run() (tgo4 error) {
	var tgo5 error = tgo4
	var tgo6 error = tgo5

	defer func() {
		if recover() != nil {
			fmt.Print("r")
		}
	}()
	var a *[2]N
	{
		var tgo1 *[2]N = a
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = step("I")
		tgo5 = tgo3
		if tgo5 != tgo6 {
			return tgo5
		}
		var tgo11 N = tgo7(&tgo1[tgo2])
		var tgo12 bool = 0 == 0

		for tgo11 = range 2 {
			if tgo12 {
				tgo12 = 0 != 0
			} else {
				tgo1 = a
				tgo2, tgo3 = step("I")
				tgo5 = tgo3
				if tgo5 != tgo6 {
					return tgo5
				}
			}
			tgo1[tgo2] = tgo11
			{

			}
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

func tgo7[tgo8 interface {
}](tgo9 *tgo8) (tgo10 tgo8) {
	return
}
