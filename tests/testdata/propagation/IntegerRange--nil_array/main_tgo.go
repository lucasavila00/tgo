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
	{
		var tgo1 *[2]N = a
		var tgo2 int
		var tgo3 error
		tgo2, tgo3 = step("I")
		if tgo3 != nil {
			return tgo3
		}
		var tgo8 N = tgo4(&tgo1[tgo2])
		var tgo9 bool = 0 == 0
		for tgo8 = range 2 {
			if tgo9 {
				tgo9 = 0 != 0
			} else {
				tgo1 = a
				tgo2, tgo3 = step("I")
				if tgo3 != nil {
					return tgo3
				}
			}
			tgo1[tgo2] = tgo8
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
func tgo4[tgo5 interface {
}](tgo6 *tgo5) (tgo7 tgo5) {
	return
}
