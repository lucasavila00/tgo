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
		if tgo3 != nil {
			tgo5 = tgo3
			return tgo5
		}
		var tgo10 N = tgo6(&tgo1[tgo2])
		var tgo11 bool = 0 == 0

		for tgo10 = range 2 {
			if tgo11 {
				tgo11 = 0 != 0
			} else {
				tgo1 = a
				tgo2, tgo3 = step("I")
				if tgo3 != nil {
					tgo5 = tgo3
					return tgo5
				}
			}
			tgo1[tgo2] = tgo10
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

func tgo6[tgo7 interface {
}](tgo8 *tgo7) (tgo9 tgo7) {
	return
}
